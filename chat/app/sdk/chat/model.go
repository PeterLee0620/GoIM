package chat

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type connection struct {
	id   uuid.UUID
	name string
	conn *websocket.Conn
}
type inMessage struct {
	FromID uuid.UUID `json:"fromID"`
	ToID   uuid.UUID `json:"toID"`
	Msg    string    `json:"msg"`
}
type outMessage struct {
	From user   `json:"from"`
	To   user   `json:"to"`
	Msg  string `json:"msg"`
}

type user struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
