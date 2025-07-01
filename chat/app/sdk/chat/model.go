package chat

import (
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type inMessage struct {
	ToID uuid.UUID `json:"toID"`
	Msg  string    `json:"msg"`
}
type outMessage struct {
	From User   `json:"from"`
	Msg  string `json:"msg"`
}

type User struct {
	ID       uuid.UUID       `json:"id"`
	Name     string          `json:"name"`
	LastPong time.Time       `json:"lastpong"`
	Conn     *websocket.Conn `json:"-"`
}

type Connection struct {
	Conn  *websocket.Conn
	Valid bool
}
