package chatapp

import (
	"net/http"

	"github.com/PeterLee0620/GoIM/business/domain/chatbus"
	"github.com/PeterLee0620/GoIM/foundation/logger"
	"github.com/PeterLee0620/GoIM/foundation/web"
)

// Routes adds specific routes for this group.
func Routes(app *web.App, log *logger.Logger, chatBus *chatbus.Business) {
	api := newApp(log, chatBus)

	app.HandlerFunc(http.MethodGet, "", "/connect", api.connect)
}
