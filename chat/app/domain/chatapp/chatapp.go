package chatapp

import (
	"context"
	"net/http"

	"github.com/DavidLee0620/GoIM/chat/foundation/web"
)

type app struct {
}

func newApp() *app {
	return &app{}
}

func (a *app) test(_ context.Context, _ *http.Request) web.Encoder {
	return status{
		Status: "ok",
	}
}
