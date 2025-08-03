package app

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/DavidLee0620/GoIM/chat/api/frontends/client/app/storage/dbfile"
	"github.com/DavidLee0620/GoIM/chat/foundation/signature"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/websocket"
)

type UIScreenWrite func(id string, msg string)
type UIUpdateContact func(id string, name string)
type Client struct {
	conn       *websocket.Conn
	url        string
	id         common.Address
	db         *dbfile.DB
	uiWrite    UIScreenWrite
	privateKey *ecdsa.PrivateKey
}

// ============================================================================
func New(id common.Address, privateKey *ecdsa.PrivateKey, url string, db *dbfile.DB) *Client {

	clt := Client{
		url:        url,
		id:         id,
		db:         db,
		privateKey: privateKey,
	}
	return &clt
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
func (c *Client) HandShake(name string, uiWrite UIScreenWrite, uiUpdateContact UIUpdateContact) error {
	conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		return fmt.Errorf("dial:%w", err)
	}
	c.conn = conn
	c.uiWrite = uiWrite
	//----------------------------------------------------------------
	//读取服务端发出的信息，若为Hello则成功
	_, msg, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read:%w", err)
	}
	if string(msg) != "Hello" {
		return fmt.Errorf("unexpected msg:%w", err)
	}
	//----------------------------------------------------------------
	//创建uuid和name的结构体，序列化后发送
	user := struct {
		ID   common.Address
		Name string
	}{
		ID:   c.id,
		Name: name,
	}
	data, err := json.Marshal(&user)
	if err != nil {
		return fmt.Errorf("json marshal:%w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("writeUI:%w", err)
	}
	//----------------------------------------------------------------
	//读取服务端发送的信息，并且打印应为Hello Lee
	_, _, err = conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read:%w", err)
	}

	//----------------------------------------------------------------
	//监听服务端的消息
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				uiWrite("system", fmt.Sprintf("read err:%s", err))
				return
			}
			var inMsg incomingMessage
			if err := json.Unmarshal(msg, &inMsg); err != nil {
				uiWrite("system", fmt.Sprintf("unmarshal err:%s", err))
				return
			}
			user, err := c.db.QueryContactByID(inMsg.From.ID)
			switch {
			case err != nil:
				user, err = c.db.InsertContact(inMsg.From.ID, inMsg.From.Name)
				if err != nil {
					uiWrite("system", fmt.Sprintf("add contact err:%s", err))
					return
				}
				uiUpdateContact(inMsg.From.ID.Hex(), inMsg.From.Name)
			default:
				inMsg.From.Name = user.Name
			}
			message := formatMessage(user.Name, inMsg.Msg)
			if err := c.db.InsertMessage(inMsg.From.ID, message); err != nil {
				uiWrite("system", fmt.Sprintf("add message err:%s", err))
				return
			}
			uiWrite(inMsg.From.ID.Hex(), message)

		}
	}()
	return nil
}

func (c *Client) Send(to common.Address, msg string) error {

	dataToSign := struct {
		ToID  common.Address
		Msg   string
		Nonce uint64
	}{
		ToID:  to,
		Msg:   msg,
		Nonce: 1,
	}
	v, r, s, err := signature.Sign(dataToSign, c.privateKey)
	if err != nil {
		return fmt.Errorf("send Sign:%w", err)
	}
	outMsg := outgoingMessage{
		ToID:  to,
		Msg:   msg,
		Nonce: 1,
		V:     v,
		R:     r,
		S:     s,
	}
	data, err := json.Marshal(&outMsg)
	if err != nil {
		return fmt.Errorf("json marshal:%w", err)
	}

	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("writeUI:%w", err)
	}

	message := formatMessage("You", msg)
	if err := c.db.InsertMessage(to, message); err != nil {
		return fmt.Errorf("add message err:%s", err)
	}
	c.uiWrite(to.Hex(), message)

	return nil
}

type outgoingMessage struct {
	ToID  common.Address `json:"toID"`
	Msg   string         `json:"msg"`
	Nonce uint64         `json:"nonce"`
	V     *big.Int       `json:"v"`
	R     *big.Int       `json:"r"`
	S     *big.Int       `json:"s"`
}
type incomingMessage struct {
	From user   `json:"from"`
	Msg  string `json:"msg"`
}

type user struct {
	ID   common.Address `json:"id"`
	Name string         `json:"name"`
}
