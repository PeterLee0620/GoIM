// Package mux provides routing support.
package mux

import (
	"context"
	"net/http"

	"github.com/PeterLee0620/GoIM/app/domain/chatapp"
	"github.com/PeterLee0620/GoIM/app/sdk/mid"
	"github.com/PeterLee0620/GoIM/business/domain/chatbus"
	"github.com/PeterLee0620/GoIM/foundation/logger"
	"github.com/PeterLee0620/GoIM/foundation/web"
)

// Config contains all the mandatory systems required by handlers.
type Config struct {
	Log        *logger.Logger
	ChatBus    *chatbus.Business
	ServerAddr string
}

// WebAPI constructs a http.Handler with all application routes bound.
func WebAPI(cfg Config) http.Handler {
	logger := func(ctx context.Context, msg string, args ...any) {
		cfg.Log.Info(ctx, msg, args...)
	}

	app := web.NewApp(
		logger,
		mid.Logger(cfg.Log),
		mid.Errors(cfg.Log),
		mid.Panics(),
	)

	chatapp.Routes(app, cfg.Log, cfg.ChatBus, cfg.ServerAddr)

	return app
}
