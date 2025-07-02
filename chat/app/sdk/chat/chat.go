// Package chat 应用层api
package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"

	"time"

	"github.com/DavidLee0620/GoIM/chat/app/sdk/errs"
	"github.com/DavidLee0620/GoIM/chat/foundation/logger"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// 错误变量
var (
	ErrNotExists = fmt.Errorf("user dosen't exists")
	ErrExists    = fmt.Errorf("user exists")
)

type Chat struct {
	log   *logger.Logger
	users Users
}

type Users interface {
	AddUser(ctx context.Context, usr User) error
	UpdateLastPong(ctx context.Context, usrID uuid.UUID) error
	RemoveUser(ctx context.Context, userID uuid.UUID)
	Connections() map[uuid.UUID]Connection
	Retrieve(ctx context.Context, userID uuid.UUID) (User, error)
}

func New(log *logger.Logger, users Users) *Chat {
	c := Chat{
		log:   log,
		users: users,
	}
	c.ping()
	return &c
}
func (c *Chat) Handshake(ctx context.Context, w http.ResponseWriter, r *http.Request) (User, error) {
	//服务端发送Hello
	var ws websocket.Upgrader
	conn, err := ws.Upgrade(w, r, nil)
	if err != nil {
		return User{}, errs.Newf(errs.FailedPrecondition, "unable to upgrade to websocket")
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte("Hello")); err != nil {
		return User{}, fmt.Errorf("write message error:%w", err)
	}
	//设置100ms的上下文
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	usr := User{
		Conn:     conn,
		LastPong: time.Now(),
	}
	//服务端读取客户端信息
	msg, err := c.readMessage(ctx, usr)
	if err != nil {
		return User{}, fmt.Errorf("read message error:%w", err)
	}
	//将接收的信息反序列化到结构体中

	if err := json.Unmarshal(msg, &usr); err != nil {
		return User{}, fmt.Errorf("unmarshal message error:%w", err)
	}
	if err := c.users.AddUser(ctx, usr); err != nil {
		defer conn.Close()
		if err := conn.WriteMessage(websocket.TextMessage, []byte("Already Connected")); err != nil {
			return User{}, fmt.Errorf("write msg:%w", err)
		}
		return User{}, fmt.Errorf("add user:%w", err)
	}
	//发送Welcome Lee到客户端
	v := fmt.Sprintf("Welcome %s", usr.Name)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(v)); err != nil {
		return User{}, fmt.Errorf("write message error:%w", err)
	}
	c.log.Info(ctx, "chat-handshake", "status", "completed", "usr", usr)
	//----------------------------------------------------------------------------
	pong := func(appData string) error {
		ctx := context.Background()
		usr, err := c.users.Retrieve(ctx, usr.ID)
		if err != nil {
			c.log.Info(ctx, "pong handler", "name", usr.Name, "id", usr.ID, "errpr", err)
			return nil
		}
		if err := c.users.UpdateLastPong(ctx, usr.ID); err != nil {
			c.log.Info(ctx, "pong handler", "name", usr.Name, "id", usr.ID, "error", err)
			return nil
		}
		return nil
	}
	ping := func(appData string) error {
		c.log.Info(ctx, "ping-handler", "name", usr.Name, "id", usr.ID)

		err := usr.Conn.WriteMessage(websocket.PongMessage, []byte("pong"))
		if err != nil {
			c.log.Info(ctx, "pong-handler", "name", usr.Name, "id", usr.ID, "error", err)

		}

		return nil
	}
	usr.Conn.SetPongHandler(pong)
	usr.Conn.SetPingHandler(ping)

	return usr, nil
}

func (c *Chat) Listen(ctx context.Context, usr User) {
	for {
		msg, err := c.readMessage(ctx, usr)
		if err != nil {
			if c.isCriticalError(ctx, err) {
				return
			}
			continue
		}

		var inMsg inMessage
		if err := json.Unmarshal(msg, &inMsg); err != nil {
			c.log.Info(ctx, "chat-listen-unmarshal", "err", err)
			continue
		}

		if err := c.sendMeessage(ctx, usr, inMsg); err != nil {
			c.log.Info(ctx, "chat-listen-send", "err", err)
		}
	}
}

// ===================================================================

func (c *Chat) readMessage(ctx context.Context, usr User) ([]byte, error) {
	type respone struct {
		msg []byte
		err error
	}
	//通过带有缓冲区的channel防止go程阻塞
	ch := make(chan respone, 1)
	go func() {
		var err error
		_, msg, err := usr.Conn.ReadMessage()
		if err != nil {
			ch <- respone{nil, err}
		}
		ch <- respone{msg, nil}
	}()
	var resp respone
	//要么超时退出，要么100ms内接收到数据退出
	select {
	case <-ctx.Done():
		c.users.RemoveUser(ctx, usr.ID)
		usr.Conn.Close()
		return nil, ctx.Err()
	case resp = <-ch:
		if resp.err != nil {
			c.users.RemoveUser(ctx, usr.ID)
			usr.Conn.Close()
			return nil, resp.err
		}
	}
	return resp.msg, nil
}

func (c *Chat) sendMeessage(ctx context.Context, usr User, msg inMessage) error {
	to, err := c.users.Retrieve(ctx, msg.ToID)
	if err != nil {
		return err
	}
	m := outMessage{
		From: User{
			ID:   usr.ID,
			Name: usr.Name,
		},
		Msg: msg.Msg,
	}

	if err := to.Conn.WriteJSON(m); err != nil {
		return fmt.Errorf("write message:%w", err)
	}

	return nil
}

func (c *Chat) ping() {
	const maxWait = 10 * time.Second
	ticker := time.NewTicker(maxWait)
	go func() {
		ctx := context.Background()
		for {

			<-ticker.C
			c.log.Info(ctx, "***ping**", "status", "strated")

			for id, conn := range c.users.Connections() {
				if time.Since(conn.LastPong) > maxWait {
					c.users.RemoveUser(ctx, id)
					continue
				}
				if err := conn.Conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
					c.log.Info(ctx, "chat-ping", "status", "failed", "id", id, "err", err)
				}
			}
			c.log.Info(ctx, "***ping**", "status", "completed")
		}

	}()
}

func (c *Chat) isCriticalError(ctx context.Context, err error) bool {
	switch e := err.(type) {
	case *websocket.CloseError:
		c.log.Info(ctx, "chat-isCriticalError", "status", "client disconnected")
		return true
	case *net.OpError:
		if !e.Temporary() {
			c.log.Info(ctx, "chat-isCriticalError", "status", "server disconnected")
			return true
		}
		return false
	default:
		if errors.Is(err, context.Canceled) {
			c.log.Info(ctx, "chat-isCriticalError", "status", "client canceled")
			return true
		}
		c.log.Info(ctx, "chat-isCriticalError", "err", err)
		return false
	}

}
