package chat

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type inMessage struct {
	FromID uuid.UUID `json:"fromID"`
	ToID   uuid.UUID `json:"toID"`
	Msg    string    `json:"msg"`
}
type outMessage struct {
	From User   `json:"from"`
	To   User   `json:"to"`
	Msg  string `json:"msg"`
}

type User struct {
	ID   uuid.UUID       `json:"id"`
	Name string          `json:"name"`
	Conn *websocket.Conn `json:"-"`
}
