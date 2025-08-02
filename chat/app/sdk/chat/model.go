package chat

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type outgoingMessage struct {
	From User   `json:"from"`
	Msg  string `json:"msg"`
}
type incomingMessage struct {
	ToID  common.Address `json:"toID"`
	Msg   string         `json:"msg"`
	Nonce uint64         `json:"nonce"`
	V     *big.Int       `json:"v"`
	R     *big.Int       `json:"r"`
	S     *big.Int       `json:"s"`
}

type User struct {
	ID       common.Address  `json:"id"`
	Name     string          `json:"name"`
	LastPing time.Time       `json:"lastping"`
	LastPong time.Time       `json:"lastpong"`
	Conn     *websocket.Conn `json:"-"`
}

type Connection struct {
	Conn     *websocket.Conn
	LastPing time.Time
	LastPong time.Time
}
type busMessage struct {
	CapID    uuid.UUID      `json:"capID"`
	FromID   common.Address `json:"from"`
	FromName string         `json:"fromName"`
	incomingMessage
}
