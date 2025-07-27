package app

import (
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
)

type UIScreenWrite func(name string, msg string)
type UIUpdateContact func(id string, name string)
type Client struct {
	conn *websocket.Conn
	url  string
	id   string
	cfg  *Config
}

// ============================================================================
func New(id string, url string, cfg *Config) *Client {

	clt := Client{
		url: url,
		id:  id,
		cfg: cfg,
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
		ID   string
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
			var outMsg outMessage
			if err := json.Unmarshal(msg, &outMsg); err != nil {
				uiWrite("system", fmt.Sprintf("unmarshal err:%s", err))
				return
			}
			user, err := c.cfg.LookupContact(outMsg.From.ID)
			switch {
			case err != nil:
				if err := c.cfg.AddContact(outMsg.From.ID, outMsg.From.Name); err != nil {
					uiWrite("system", fmt.Sprintf("add contact err:%s", err))
					return
				}
				uiUpdateContact(outMsg.From.ID, outMsg.From.Name)
			default:
				outMsg.From.Name = user.Name
			}
			uiWrite(outMsg.From.Name, outMsg.Msg)

		}
	}()
	return nil
}

func (c *Client) Send(to string, msg string) error {
	inMsg := inMessage{
		ToID: to,
		Msg:  msg,
	}
	data2, err := json.Marshal(&inMsg)
	if err != nil {
		return fmt.Errorf("json marshal:%w", err)
	}

	if err := c.conn.WriteMessage(websocket.TextMessage, data2); err != nil {
		return fmt.Errorf("writeUI:%w", err)
	}
	return nil
}

type inMessage struct {
	ToID string `json:"toID"`
	Msg  string `json:"msg"`
}
type outMessage struct {
	From user   `json:"from"`
	Msg  string `json:"msg"`
}

type user struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
