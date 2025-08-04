// Package app provides client app support.
package app

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/DavidLee0620/GoIM/chat/foundation/signature"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/websocket"
)

type MyAccount struct {
	ID   common.Address
	Name string
}

type User struct {
	ID           common.Address
	Name         string
	AppLastNonce uint64
	LastNonce    uint64
	Messages     []string
}

type Storage interface {
	QueryContactByID(id common.Address) (User, error)
	InsertContact(id common.Address, name string) (User, error)
	InsertMessage(id common.Address, msg string) error
	UpdateAppNonce(id common.Address, nonce uint64) error
}

type UI interface {
	Run() error
	WriteText(id string, msg string)
	UpdateContact(id string, name string)
}

// =============================================================================

type outgoingMessage struct {
	ToID      common.Address `json:"toID"`
	Msg       string         `json:"msg"`
	FromNonce uint64         `json:"fromNonce"`
	V         *big.Int       `json:"v"`
	R         *big.Int       `json:"r"`
	S         *big.Int       `json:"s"`
}

type usr struct {
	ID    common.Address `json:"id"`
	Name  string         `json:"name"`
	Nonce uint64         `json:"nonce"`
}

type incomingMessage struct {
	From usr    `json:"from"`
	Msg  string `json:"msg"`
}

// =============================================================================

type App struct {
	db          Storage
	ui          UI
	myAccountID common.Address
	privateKey  *ecdsa.PrivateKey
	url         string
	conn        *websocket.Conn
}

func NewApp(db Storage, ui UI, myAccountID common.Address, privateKey *ecdsa.PrivateKey, url string) *App {
	return &App{
		db:          db,
		ui:          ui,
		myAccountID: myAccountID,
		privateKey:  privateKey,
		url:         url,
	}
}

func (app *App) Close() error {
	if app.conn == nil {
		return nil
	}

	return app.conn.Close()
}

func (app *App) Run() error {
	return app.ui.Run()
}

func (app *App) Handshake(acct MyAccount) error {
	conn, _, err := websocket.DefaultDialer.Dial(app.url, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	app.conn = conn

	// -------------------------------------------------------------------------

	_, msg, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	if string(msg) != "HELLO" {
		return fmt.Errorf("unexpected message: %s", msg)
	}

	// -------------------------------------------------------------------------

	user := struct {
		ID   common.Address
		Name string
	}{
		ID:   app.myAccountID,
		Name: acct.Name,
	}

	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	// -------------------------------------------------------------------------

	if _, _, err = conn.ReadMessage(); err != nil {
		return fmt.Errorf("read: %w", err)
	}

	// -------------------------------------------------------------------------

	go func() {
		app.ReceiveCapMessage(conn)
	}()

	return nil
}

// =============================================================================

func (app *App) ReceiveCapMessage(conn *websocket.Conn) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			app.ui.WriteText("system", fmt.Sprintf("read: %s", err))
			return
		}

		var inMsg incomingMessage
		if err := json.Unmarshal(msg, &inMsg); err != nil {
			app.ui.WriteText("system", fmt.Sprintf("unmarshal: %s", err))
			return
		}

		user, err := app.db.QueryContactByID(inMsg.From.ID)
		switch {
		case err != nil:
			user, err = app.db.InsertContact(inMsg.From.ID, inMsg.From.Name)
			if err != nil {
				app.ui.WriteText("system", fmt.Sprintf("add contact: %s", err))
				return
			}

			app.ui.UpdateContact(inMsg.From.ID.Hex(), inMsg.From.Name)

		default:
			inMsg.From.Name = user.Name
		}

		// -----------------------------------------------------------------

		expNonce := user.LastNonce + 1
		if inMsg.From.Nonce != expNonce {
			app.ui.WriteText("system", fmt.Sprintf("invalid nonce: %d != %d", inMsg.From.Nonce, expNonce))
			return
		}

		if err := app.db.UpdateAppNonce(inMsg.From.ID, expNonce); err != nil {
			app.ui.WriteText("system", fmt.Sprintf("update app nonce: %s", err))
			return
		}

		// -----------------------------------------------------------------

		fm := formatMessage(user.Name, inMsg.Msg)

		if err := app.db.InsertMessage(inMsg.From.ID, fm); err != nil {
			app.ui.WriteText("system", fmt.Sprintf("add message: %s", err))
			return
		}

		app.ui.WriteText(inMsg.From.ID.Hex(), fm)
	}
}

func (app *App) SendMessageHandler(to common.Address, msg string) error {
	if app.conn == nil {
		return fmt.Errorf("no connection")
	}

	usr, err := app.db.QueryContactByID(to)
	if err != nil {
		return fmt.Errorf("query contact: %w", err)
	}

	nonce := usr.AppLastNonce + 1

	// -------------------------------------------------------------------------

	dataToSign := struct {
		ToID      common.Address
		Msg       string
		FromNonce uint64
	}{
		ToID:      to,
		Msg:       msg,
		FromNonce: nonce,
	}

	v, r, s, err := signature.Sign(dataToSign, app.privateKey)
	if err != nil {
		return fmt.Errorf("signing: %w", err)
	}

	outMsg := outgoingMessage{
		ToID:      to,
		Msg:       msg,
		FromNonce: nonce,
		V:         v,
		R:         r,
		S:         s,
	}

	data, err := json.Marshal(outMsg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if err := app.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	// -------------------------------------------------------------------------

	if err := app.db.UpdateAppNonce(to, nonce); err != nil {
		return fmt.Errorf("update app nonce: %w", err)
	}

	msg = formatMessage("You", msg)

	if err := app.db.InsertMessage(to, msg); err != nil {
		return fmt.Errorf("add message: %w", err)
	}

	// -------------------------------------------------------------------------

	app.ui.WriteText(to.Hex(), msg)

	return nil
}
