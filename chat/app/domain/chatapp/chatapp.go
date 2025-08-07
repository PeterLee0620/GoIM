// Package chatapp provides the application layer for the chat service.
package chatapp

import (
	"context"
	"net/http"

	"github.com/DavidLee0620/GoIM/chat/app/sdk/errs"
	"github.com/DavidLee0620/GoIM/chat/business/chatbus"
	"github.com/DavidLee0620/GoIM/chat/foundation/logger"
	"github.com/DavidLee0620/GoIM/chat/foundation/web"
)

type app struct {
	log  *logger.Logger
	chat *chatbus.Chat
}

func newApp(log *logger.Logger, chat *chatbus.Chat) *app {
	return &app{
		log:  log,
		chat: chat,
	}
}

func (a *app) connect(ctx context.Context, r *http.Request) web.Encoder {
	usr, err := a.chat.Handshake(ctx, web.GetWriter(ctx), r)
	if err != nil {
		return errs.Newf(errs.FailedPrecondition, "handshake failed: %s", err)
	}
	defer usr.Conn.Close()

	a.chat.ListenClient(ctx, usr)

	return web.NewNoResponse()
}
