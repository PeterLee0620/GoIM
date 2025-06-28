package chat

import (
	"context"
	"encoding/json"
	"fmt"

	"time"

	"github.com/DavidLee0620/GoIM/chat/foundation/logger"
	"github.com/gorilla/websocket"
)

type Chat struct {
	log *logger.Logger
}

func New(log *logger.Logger) *Chat {
	return &Chat{
		log: log,
	}
}

func (c *Chat) Handshake(ctx context.Context, conn *websocket.Conn) (User, error) {
	//服务端发送Hello
	if err := conn.WriteMessage(websocket.TextMessage, []byte("Hello")); err != nil {
		return User{}, fmt.Errorf("write message error:%w", err)
	}
	//设置100ms的上下文
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	//服务端读取客户端信息
	msg, err := c.readMessage(ctx, conn)
	if err != nil {
		return User{}, fmt.Errorf("read message error:%w", err)
	}
	//将接收的信息反序列化到结构体中
	var use User
	if err := json.Unmarshal(msg, &use); err != nil {
		return User{}, fmt.Errorf("unmarshal message error:%w", err)
	}
	//发送Welcome Lee到客户端
	v := fmt.Sprintf("Welcome %s", use.Name)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(v)); err != nil {
		return User{}, fmt.Errorf("write message error:%w", err)
	}
	return use, nil
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
