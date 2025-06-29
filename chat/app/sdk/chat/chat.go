package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"time"

	"github.com/DavidLee0620/GoIM/chat/app/sdk/errs"
	"github.com/DavidLee0620/GoIM/chat/foundation/logger"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var ErrFromNotExists = fmt.Errorf("from user dosen't exists")
var ErrToNotExists = fmt.Errorf("to user dosen't exists")

type Chat struct {
	log   *logger.Logger
	users map[uuid.UUID]User
	mu    sync.RWMutex
}

func New(log *logger.Logger) *Chat {
	c := Chat{
		log:   log,
		users: make(map[uuid.UUID]User),
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
		Conn: conn,
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
	if err := c.addUser(ctx, usr); err != nil {
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
	c.log.Info(ctx, "handshake completed", "usr", usr)
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
			c.log.Info(ctx, "listen-unmarshal", "err", err)
			continue
		}

		if err := c.sendMeessage(inMsg); err != nil {
			c.log.Info(ctx, "listen-send", "err", err)
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
		c.log.Info(ctx, "chat-readMessage", "status", "started")
		defer c.log.Info(ctx, "chat-readMessage", "status", "completed")
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
		c.removeUser(ctx, usr.ID)
		return nil, ctx.Err()
	case resp = <-ch:
		if resp.err != nil {
			c.removeUser(ctx, usr.ID)
			return nil, resp.err
		}
	}
	return resp.msg, nil
}

func (c *Chat) sendMeessage(msg inMessage) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	from, exists := c.users[msg.FromID]
	if !exists {
		return ErrFromNotExists
	}
	to, exists := c.users[msg.ToID]
	if !exists {
		return ErrToNotExists
	}
	m := outMessage{
		From: User{
			ID:   from.ID,
			Name: from.Name,
		},
		To: User{
			ID:   to.ID,
			Name: to.Name,
		},
		Msg: msg.Msg,
	}

	if err := to.Conn.WriteJSON(m); err != nil {
		return fmt.Errorf("write message:%w", err)
	}

	return nil
}
func (c *Chat) addUser(ctx context.Context, usr User) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.users[usr.ID]; exists {
		return fmt.Errorf("user exists")
	}
	c.log.Info(ctx, "add user", "name", usr.Name, "id", usr.ID)

	c.users[usr.ID] = usr
	return nil
}
func (c *Chat) removeUser(ctx context.Context, userID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	usr, exists := c.users[userID]
	if !exists {
		c.log.Info(ctx, "remove user", "userID", userID, "doesn't exisrs")
		return
	}
	c.log.Info(ctx, "remove user", "name", usr.Name, "id", usr.ID)
	delete(c.users, userID)
	usr.Conn.Close()
}

func (c *Chat) connections() map[uuid.UUID]*websocket.Conn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	m := make(map[uuid.UUID]*websocket.Conn)
	for id, usr := range c.users {
		m[id] = usr.Conn
	}
	return m
}

func (c *Chat) ping() {

	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			ctx := context.Background()
			<-ticker.C

			for k, conn := range c.connections() {

				if err := conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
					c.log.Info(ctx, "ping", "status", "failed", "id", k, "err", err)
				}
			}

		}
	}()
}

func (c *Chat) isCriticalError(ctx context.Context, err error) bool {
	switch err.(type) {
	case *websocket.CloseError:
		c.log.Info(ctx, "listen-read", "status", "client disconnected")
		return true
	default:
		if errors.Is(err, context.Canceled) {
			c.log.Info(ctx, "listen-read", "status", "client canceled")
			return true
		}
		c.log.Info(ctx, "listen-read", "err", err)
		return false
	}

}
