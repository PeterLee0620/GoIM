// Package mux provides routing support.
package mux

import (
	"context"
	"net/http"

	"github.com/DavidLee0620/GoIM/chat/app/domain/chatapp"
	"github.com/DavidLee0620/GoIM/chat/app/sdk/chat"
	"github.com/DavidLee0620/GoIM/chat/app/sdk/mid"
	"github.com/DavidLee0620/GoIM/chat/foundation/logger"
	"github.com/DavidLee0620/GoIM/chat/foundation/web"
)

// Config contains all the mandatory systems required by handlers.
type Config struct {
	Log  *logger.Logger
	Chat *chat.Chat
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

	chatapp.Routes(app, cfg.Log, cfg.Chat)

	return app
}
