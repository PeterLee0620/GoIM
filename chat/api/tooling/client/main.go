package main

import (
	"fmt"
	"os"

	"github.com/DavidLee0620/GoIM/chat/api/tooling/client/app"
	"github.com/google/uuid"
)

const (
	url            = "ws://localhost:3000/connect"
	configFilePath = "chat/zarf/"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("Error:%s\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := app.NewConfig(configFilePath)
	if err != nil {
		return fmt.Errorf("error config:%w", err)
	}
	id := cfg.User().ID
	name := cfg.User().Name
	client := app.New(id, url)
	defer client.Close()
	app := app.NewApp(client, cfg)
	log := func(name string, msg string) {
		app.WriteText(name, msg)
	}
	if err := client.HandShake(name, log); err != nil {
		return fmt.Errorf("error HandShake:%w", err)
	}
	app.WriteText("system", "CONNECTED")
	if err := app.Run(); err != nil {
		return fmt.Errorf("error running app:%w", err)
	}
	return nil
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
