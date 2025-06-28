package chatapp

import (
	"context"
	"net/http"

	"github.com/DavidLee0620/GoIM/chat/app/sdk/chat"
	"github.com/DavidLee0620/GoIM/chat/app/sdk/errs"
	"github.com/DavidLee0620/GoIM/chat/foundation/logger"
	"github.com/DavidLee0620/GoIM/chat/foundation/web"
	"github.com/gorilla/websocket"
)

type app struct {
	log  *logger.Logger
	WS   websocket.Upgrader
	chat *chat.Chat
}

func newApp(log *logger.Logger) *app {
	return &app{
		log:  log,
		chat: chat.New(log),
	}
}

func (a *app) connect(ctx context.Context, r *http.Request) web.Encoder {
	//创建websocket的握手连接
	c, err := a.WS.Upgrade(web.GetWriter(ctx), r, nil)
	if err != nil {
		return errs.Newf(errs.FailedPrecondition, "unable to upgrade to websocket")
	}
	defer c.Close()

	if err := a.chat.Handshake(ctx, c); err != nil {
		return errs.Newf(errs.FailedPrecondition, "handshake failed:%s", err)
	}
	a.chat.Listen(ctx, c)
	return web.NewNoResponse()
}
