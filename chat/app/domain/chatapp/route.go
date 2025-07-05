package chatapp

import (
	"net/http"

	"github.com/DavidLee0620/GoIM/chat/app/sdk/chat"
	"github.com/DavidLee0620/GoIM/chat/foundation/logger"
	"github.com/DavidLee0620/GoIM/chat/foundation/web"
)

func Routes(app *web.App, log *logger.Logger, chat *chat.Chat) {
	api := newApp(log, chat)
	app.HandlerFunc(http.MethodGet, "", "/connect", api.connect)
}
