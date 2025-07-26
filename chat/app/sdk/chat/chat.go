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
	"github.com/DavidLee0620/GoIM/chat/foundation/web"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// 错误变量
var (
	ErrNotExists = fmt.Errorf("user dosen't exists")
	ErrExists    = fmt.Errorf("user exists")
)

type Chat struct {
	log      *logger.Logger
	js       jetstream.JetStream
	consumer jetstream.Consumer
	subject  string
	users    Users
	stream   jetstream.Stream
	capID    uuid.UUID
}

type Users interface {
	AddUser(ctx context.Context, usr User) error
	UpdateLastPing(ctx context.Context, usrID string) error
	UpdateLastPong(ctx context.Context, usrID string) (User, error)
	RemoveUser(ctx context.Context, userID string)
	Connections() map[string]Connection
	Retrieve(ctx context.Context, userID string) (User, error)
}

func New(log *logger.Logger, conn *nats.Conn, subject string, users Users, capID uuid.UUID) (*Chat, error) {
	ctx := context.TODO()
	js, err := jetstream.New(conn)
	if err != nil {
		return nil, fmt.Errorf("nats new js: %w", err)
	}

	s1, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     subject,
		Subjects: []string{subject},
	})
	if err != nil {
		return nil, fmt.Errorf("nats create js: %w", err)
	}
	c1, err := s1.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       capID.String(),
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverNewPolicy,
	})
	if err != nil {
		return nil, fmt.Errorf("nats create Consumer:%w", err)
	}

	c := Chat{
		log:      log,
		users:    users,
		subject:  subject,
		consumer: c1,
		js:       js,
		stream:   s1,
		capID:    capID,
	}

	c1.Consume(c.listenBus(), jetstream.PullMaxMessages(1))
	const maxWait = 10 * time.Second
	c.ping(maxWait)

	return &c, nil
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
		LastPing: time.Now(),
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
	//------------------------------------------------------------------------
	if err := c.users.AddUser(ctx, usr); err != nil {
		defer conn.Close()
		if err := conn.WriteMessage(websocket.TextMessage, []byte("Already Connected")); err != nil {
			return User{}, fmt.Errorf("write msg:%w", err)
		}
		return User{}, fmt.Errorf("add user:%w", err)
	}

	usr.Conn.SetPongHandler(c.pong(usr.ID))
	//------------------------------------------------------------------------

	//发送Welcome Lee到客户端
	v := fmt.Sprintf("Welcome %s", usr.Name)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(v)); err != nil {
		return User{}, fmt.Errorf("write message error:%w", err)
	}
	c.log.Info(ctx, "chat-handshake", "status", "completed", "usr", usr)
	//----------------------------------------------------------------------------

	return usr, nil
}

func (c *Chat) ListenSocket(ctx context.Context, from User) {
	for {
		msg, err := c.readMessage(ctx, from)
		if err != nil {
			if c.isCriticalError(ctx, err) {
				return
			}
			continue
		}

		var inMsg inMessage
		if err := json.Unmarshal(msg, &inMsg); err != nil {
			c.log.Info(ctx, "log-listen-unmarshal", "err", err)
			continue
		}
		c.log.Info(ctx, "LOC:msg recv", "from", from.ID, "to", inMsg.ToID, "message", inMsg.Msg)

		to, err := c.users.Retrieve(ctx, inMsg.ToID)
		if err != nil {
			switch {
			case errors.Is(err, ErrNotExists):
				if err := c.sendMessageBus(ctx, from, inMsg); err != nil {
					c.log.Info(ctx, "loc-sendMessageBus", "ERROR", err)
				}
				c.log.Info(ctx, "loc-retrieve", "status", "user not found,sending over bus")
			default:
				c.log.Info(ctx, "loc-retrieve", "ERROR", err)

			}
			continue

		}
		if err := c.sendMeessage(from, to, inMsg.Msg); err != nil {
			c.log.Info(ctx, "log-listen-send", "err", err)
		}
		c.log.Info(ctx, "LOC:msg sent", "from", from.ID, "to", inMsg.ToID)

	}
}

// ===================================================================

func (c *Chat) listenBus() func(msg jetstream.Msg) {
	ctx := web.SetTraceID(context.Background(), uuid.New())

	f := func(msg jetstream.Msg) {
		var busMsg busMessage
		if err := json.Unmarshal(msg.Data(), &busMsg); err != nil {
			c.log.Info(ctx, "bus-listen-unmarshal", "err", err)
			return
		}
		if busMsg.CapID == c.capID {
			return
		}
		c.log.Info(ctx, "BUS:msg recv", "from", busMsg.FromID, "to", busMsg.ToID, "message", busMsg.Msg)

		to, err := c.users.Retrieve(ctx, busMsg.ToID)
		if err != nil {
			switch {
			case errors.Is(err, ErrNotExists):
				c.log.Info(ctx, "bus-retrieve", "status", "user not found")
			default:
				c.log.Info(ctx, "bus-retrieve", "ERROR", err)

			}
			return
		}
		from := User{
			ID:   busMsg.FromID,
			Name: busMsg.FromName,
		}
		if err := c.sendMeessage(from, to, busMsg.Msg); err != nil {
			c.log.Info(ctx, "bus-listen-send", "err", err)
		}
		msg.Ack()
		c.log.Info(ctx, "BUS:msg sent", "from", busMsg.FromID, "to", busMsg.ToID)

	}
	return f
}
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

func (c *Chat) sendMessageBus(ctx context.Context, from User, inMsg inMessage) error {
	busMsg := busMessage{
		FromID:   from.ID,
		FromName: from.Name,
		ToID:     inMsg.ToID,
		Msg:      inMsg.Msg,
		CapID:    c.capID,
	}
	d, err := json.Marshal(busMsg)
	if err != nil {
		return fmt.Errorf("SendToBus- marshal message: %w", err)
	}
	_, err = c.js.Publish(ctx, c.subject, d)
	if err != nil {
		return fmt.Errorf("SendToBus- publish: %w", err)
	}
	return nil
}
func (c *Chat) sendMeessage(from User, to User, msg string) error {

	m := outMessage{
		From: User{
			ID:   from.ID,
			Name: from.Name,
		},
		Msg: msg,
	}

	if err := to.Conn.WriteJSON(m); err != nil {
		return fmt.Errorf("write message:%w", err)
	}

	return nil
}

func (c *Chat) ping(maxWait time.Duration) {

	ticker := time.NewTicker(maxWait)
	go func() {

		ctx := web.SetTraceID(context.Background(), uuid.New())
		for {

			<-ticker.C

			for id, conn := range c.users.Connections() {
				sub := conn.LastPong.Sub(conn.LastPing)
				if sub > maxWait {
					c.log.Info(ctx, "***ping**", "ping", conn.LastPing.String(), "pong", conn.LastPong.String(), "sub", sub.String())
					c.users.RemoveUser(ctx, id)
					continue
				}
				c.log.Debug(ctx, "***ping**", "status", "sending")
				if err := conn.Conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
					c.log.Info(ctx, "chat-ping", "status", "failed", "id", id, "err", err)
				}

				if err := c.users.UpdateLastPing(ctx, id); err != nil {
					c.log.Info(ctx, "***ping***", "status", "failed", "id", id, "err", err)
				}
			}

		}

	}()
}
func (c *Chat) pong(id string) func(appData string) error {
	f := func(appData string) error {
		ctx := web.SetTraceID(context.Background(), uuid.New())
		usr, err := c.users.UpdateLastPong(ctx, id)
		if err != nil {
			c.log.Info(ctx, "***pong***", "id", id, "error", err)
			return nil
		}

		sub := usr.LastPong.Sub(usr.LastPing)
		c.log.Debug(ctx, "***pong**", "id", id, "status", "received", "sub", sub.String())

		return nil
	}
	return f
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
		if errors.Is(err, nats.ErrConnectionClosed) {
			c.log.Info(ctx, "chat-isCriticalError", "status", "nats connection canceled")
			return true
		}

		c.log.Info(ctx, "chat-isCriticalError", "err", err)
		return false
	}

}
