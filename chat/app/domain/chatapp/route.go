package chatapp

import (
	"net/http"

	"github.com/DavidLee0620/GoIM/chat/app/sdk/chat"
	"github.com/DavidLee0620/GoIM/chat/app/sdk/chat/users"
	"github.com/DavidLee0620/GoIM/chat/foundation/logger"
	"github.com/DavidLee0620/GoIM/chat/foundation/web"
)

func Routes(app *web.App, log *logger.Logger) {
	api := newApp(log, chat.New(log, users.New(log)))
	app.HandlerFunc(http.MethodGet, "", "/connect", api.connect)
}
