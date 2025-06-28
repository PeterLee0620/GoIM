package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"time"

	"github.com/DavidLee0620/GoIM/chat/foundation/logger"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var ErrFromNotExists = fmt.Errorf("from user dosen't exists")
var ErrToNotExists = fmt.Errorf("to user dosen't exists")

type Chat struct {
	log   *logger.Logger
	users map[uuid.UUID]connection
	mu    sync.RWMutex
}

func New(log *logger.Logger) *Chat {
	c := Chat{
		log:   log,
		users: make(map[uuid.UUID]connection),
	}
	c.ping()
	return &c
}
func (c *Chat) Handshake(ctx context.Context, conn *websocket.Conn) error {
	//服务端发送Hello
	if err := conn.WriteMessage(websocket.TextMessage, []byte("Hello")); err != nil {
		return fmt.Errorf("write message error:%w", err)
	}
	//设置100ms的上下文
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	//服务端读取客户端信息
	msg, err := c.readMessage(ctx, conn)
	if err != nil {
		return fmt.Errorf("read message error:%w", err)
	}
	//将接收的信息反序列化到结构体中
	var usr user
	if err := json.Unmarshal(msg, &usr); err != nil {
		return fmt.Errorf("unmarshal message error:%w", err)
	}
	if err := c.addUser(usr, conn); err != nil {
		defer conn.Close()
		if err := conn.WriteMessage(websocket.TextMessage, []byte("Already Connected")); err != nil {
			return fmt.Errorf("write msg:%w", err)
		}
		return fmt.Errorf("add user:%w", err)
	}
	//发送Welcome Lee到客户端
	v := fmt.Sprintf("Welcome %s", usr.Name)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(v)); err != nil {
		return fmt.Errorf("write message error:%w", err)
	}
	c.log.Info(ctx, "handshake completed", "usr", usr)
	return nil
}

func (c *Chat) Listen(ctx context.Context, conn *websocket.Conn) {
	for {
		msg, err := c.readMessage(ctx, conn)
		if err != nil {
			c.log.Info(ctx, "listen-read", "err", err)
			return
		}
		var inMsg inMessage
		if err := json.Unmarshal(msg, &inMsg); err != nil {
			c.log.Info(ctx, "listen-unmarshal", "err", err)
			return
		}

		if err := c.sendMeessage(inMsg); err != nil {
			c.log.Info(ctx, "listen-send", "err", err)
		}
	}
}

// ===================================================================
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
		From: user{
			ID:   from.id,
			Name: from.name,
		},
		To: user{
			ID:   to.id,
			Name: to.name,
		},
		Msg: msg.Msg,
	}

	if err := to.conn.WriteJSON(m); err != nil {
		return fmt.Errorf("write message:%w", err)
	}

	return nil
}
func (c *Chat) addUser(usr user, conn *websocket.Conn) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.users[usr.ID]; exists {
		return fmt.Errorf("user exists")
	}
	c.log.Info(context.Background(), "remove user", "name", usr.Name, "id", usr.ID)

	c.users[usr.ID] = connection{
		id:   usr.ID,
		name: usr.Name,
		conn: conn,
	}
	return nil
}
func (c *Chat) removeUser(userID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, exists := c.users[userID]
	if !exists {
		c.log.Info(context.Background(), "remove user", "userID", userID, "doesn't exisrs")
		return
	}
	c.log.Info(context.Background(), "remove user", "name", v.name, "id", v.id)
	delete(c.users, userID)
	v.conn.Close()
}

func (c *Chat) readMessage(ctx context.Context, conn *websocket.Conn) ([]byte, error) {
	type respone struct {
		msg []byte
		err error
	}
	//通过带有缓冲区的channel防止go程阻塞
	ch := make(chan respone, 1)
	go func() {
		c.log.Info(ctx, "chat", "status", "starting handshake read")
		defer c.log.Info(ctx, "chat", "status", "completed handshake read")
		_, msg, err := conn.ReadMessage()
		if err != nil {
			ch <- respone{nil, err}
		}
		ch <- respone{msg, nil}
	}()
	var resp respone
	//要么超时退出，要么100ms内接收到数据退出
	select {
	case <-ctx.Done():
		conn.Close()
		return nil, ctx.Err()
	case resp = <-ch:
		if resp.err != nil {
			return nil, fmt.Errorf("empty message")
		}
	}
	return resp.msg, nil
}

func (c *Chat) connections() map[uuid.UUID]connection {
	c.mu.RLock()
	defer c.mu.RUnlock()
	m := make(map[uuid.UUID]connection)
	for k, v := range c.users {
		m[k] = v
	}
	return m
}

func (c *Chat) ping() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			<-ticker.C
			c.log.Info(context.Background(), "ping", "status", "started")
			for k, v := range c.connections() {
				c.log.Info(context.Background(), "ping", "name", v.name, "id", v.id)

				if err := v.conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
					c.removeUser(k)
				}
			}
			c.log.Info(context.Background(), "ping", "status", "completed")
		}
	}()
}
