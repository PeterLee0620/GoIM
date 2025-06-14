package chatapp

import (
	"net/http"

	"github.com/DavidLee0620/GoIM/chat/foundation/web"
)

func Routes(app *web.App) {
	api := newApp()
	app.HandlerFunc(http.MethodGet, "", "/test", api.test)
}
