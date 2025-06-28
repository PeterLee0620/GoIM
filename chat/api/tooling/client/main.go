package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func main() {
	if err := hack1(); err != nil {
		log.Fatal(err)
	}
}

func hack1() error {
	//创建websocket连接
	const url = "ws://localhost:3000/connect"
	req := make(http.Header)
	socket, _, err := websocket.DefaultDialer.Dial(url, req)
	if err != nil {
		return fmt.Errorf("dial:%w", err)
	}
	defer socket.Close()
	//----------------------------------------------------------------
	//读取服务端发出的信息，若为Hello则成功
	_, msg, err := socket.ReadMessage()
	if err != nil {
		return fmt.Errorf("read:%w", err)
	}
	if string(msg) != "Hello" {
		return fmt.Errorf("unexpected msg:%w", err)
	}
	//----------------------------------------------------------------
	//创建uuid和name的结构体，序列化后发送
	user := struct {
		ID   uuid.UUID
		Name string
	}{
		ID:   uuid.New(),
		Name: "Lee",
	}
	data, err := json.Marshal(&user)
	if err != nil {
		return fmt.Errorf("json marshal:%w", err)
	}

	if err := socket.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("write:%w", err)
	}
	//----------------------------------------------------------------
	//读取服务端发送的信息，并且打印应为Hello Lee
	_, msg, err = socket.ReadMessage()
	if err != nil {
		return fmt.Errorf("read:%w", err)
	}
	fmt.Println(string(msg))
	return nil
}
