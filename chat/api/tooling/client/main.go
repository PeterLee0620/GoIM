package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/DavidLee0620/GoIM/chat/api/tooling/client/chat"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func main() {
	app := chat.NewApp()
	const url = "ws://localhost:3000/connect"
	chat.NewClient(app, ID, url)
	app.SetButtonHandler()
	app.WriteText("This is a test func")
	if err := app.Run(); err != nil {
		fmt.Printf("Error running app:%s", err)
		os.Exit(1)
	}
}

func hack1() error {
	//创建websocket连接
	const url = "ws://localhost:3000/connect"
	req := make(http.Header)
	users := []uuid.UUID{
		uuid.MustParse("f3cf4d43-9585-4398-8613-0a5787b1aede"),
		uuid.MustParse("c60e6de5-3b1d-4500-aba8-ca89903118d0"),
	}
	var ID uuid.UUID
	switch os.Args[1] {
	case "0":
		ID = users[0]
	case "1":
		ID = users[1]
	}
	fmt.Println("ID:", ID)
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
		ID:   ID,
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

	//----------------------------------------------------------------
	//监听服务端的消息
	go func() {
		for {
			_, msg, err := socket.ReadMessage()
			if err != nil {
				fmt.Printf("read err:%s", err)
				return
			}
			var outMsg outMessage
			if err := json.Unmarshal(msg, &outMsg); err != nil {
				fmt.Printf("unmarshal err:%s", err)
				return
			}

			fmt.Printf("\n%s\n", outMsg.Msg)

		}
	}()

	//----------------------------------------------------------------
	for {
		fmt.Print("\n\n")
		fmt.Println("message >")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error input:%w", err)
		}

		var to uuid.UUID
		switch os.Args[1] {
		case "0":
			to = users[1]
		case "1":
			to = users[0]
		}
		inMsg := inMessage{
			ToID: to,
			Msg:  input,
		}
		data2, err := json.Marshal(&inMsg)
		if err != nil {
			return fmt.Errorf("json marshal:%w", err)
		}

		if err := socket.WriteMessage(websocket.TextMessage, data2); err != nil {
			return fmt.Errorf("write:%w", err)
		}
	}

}

type inMessage struct {
	ToID uuid.UUID `json:"toID"`
	Msg  string    `json:"msg"`
}
type outMessage struct {
	From user   `json:"from"`
	Msg  string `json:"msg"`
}

type user struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
