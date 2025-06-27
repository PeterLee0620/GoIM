package chatapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/DavidLee0620/GoIM/chat/app/sdk/errs"
	"github.com/DavidLee0620/GoIM/chat/foundation/logger"
	"github.com/DavidLee0620/GoIM/chat/foundation/web"
	"github.com/gorilla/websocket"
)

type app struct {
	log *logger.Logger
	WS  websocket.Upgrader
}

func newApp(log *logger.Logger) *app {
	return &app{
		log: log,
	}
}

func (a *app) connect(ctx context.Context, r *http.Request) web.Encoder {
	c, err := a.WS.Upgrade(web.GetWriter(ctx), r, nil)
	if err != nil {
		return errs.Newf(errs.FailedPrecondition, "unable to upgrade to websocket")
	}
	defer c.Close()
	usr, err := a.handshake(c)
	if err != nil {
		return errs.Newf(errs.FailedPrecondition, "unable to handshake:%s", err)
	}
	a.log.Info(ctx, "handshake completed", "usr", usr)
	return web.NewNoResponse()
}

func (a *app) handshake(c *websocket.Conn) (user, error) {
	if err := c.WriteMessage(websocket.TextMessage, []byte("Hello")); err != nil {
		return user{}, fmt.Errorf("write message error:%w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	msg, err := a.readMessage(ctx, c)
	if err != nil {
		return user{}, fmt.Errorf("read message error:%w", err)
	}
	var use user
	if err := json.Unmarshal(msg, &use); err != nil {
		return user{}, fmt.Errorf("unmarshal message error:%w", err)
	}
	v := fmt.Sprintf("Welcome %s", use.Name)
	if err := c.WriteMessage(websocket.TextMessage, []byte(v)); err != nil {
		return user{}, fmt.Errorf("write message error:%w", err)
	}
	return use, nil
}

func (a *app) readMessage(ctx context.Context, c *websocket.Conn) ([]byte, error) {
	type respone struct {
		msg []byte
		err error
	}
	ch := make(chan respone, 1)
	go func() {
		a.log.Info(ctx, "starting handshake read")
		defer a.log.Info(ctx, "completed handshake read")
		_, msg, err := c.ReadMessage()
		if err != nil {
			ch <- respone{nil, err}
		}
		ch <- respone{msg, nil}
	}()
	var resp respone
	select {
	case <-ctx.Done():
		c.Close()
		return nil, ctx.Err()
	case resp = <-ch:
		if resp.err != nil {
			return nil, fmt.Errorf("empty message")
		}
	}
	return resp.msg, nil
}
