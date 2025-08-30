package chatapp

import (
	"net/http"

	"github.com/PeterLee0620/GoIM/business/domain/chatbus"
	"github.com/PeterLee0620/GoIM/foundation/logger"
	"github.com/PeterLee0620/GoIM/foundation/web"
)

// Routes adds specific routes for this group.
func Routes(app *web.App, log *logger.Logger, chatBus *chatbus.Business, serverAddr string) {
	api := newApp(log, chatBus, serverAddr)

	app.HandlerFunc(http.MethodGet, "", "/connect", api.connect)
	app.HandlerFunc(http.MethodPost, "", "/tcpconnect", api.tcpConnect)
}
