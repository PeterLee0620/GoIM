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
	//创建websocket的握手连接
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
	//服务端发送Hello
	if err := c.WriteMessage(websocket.TextMessage, []byte("Hello")); err != nil {
		return user{}, fmt.Errorf("write message error:%w", err)
	}
	//设置100ms的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	//服务端读取客户端信息
	msg, err := a.readMessage(ctx, c)
	if err != nil {
		return user{}, fmt.Errorf("read message error:%w", err)
	}
	//将接收的信息反序列化到结构体中
	var use user
	if err := json.Unmarshal(msg, &use); err != nil {
		return user{}, fmt.Errorf("unmarshal message error:%w", err)
	}
	//发送Welcome Lee到客户端
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
	//通过带有缓冲区的channel防止go程阻塞
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
	//要么超时退出，要么100ms内接收到数据退出
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
