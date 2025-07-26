package main

import (
	"fmt"
	"os"

	"github.com/DavidLee0620/GoIM/chat/api/tooling/client/app"
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
	client := app.New(id, url, cfg)
	defer client.Close()
	a := app.NewApp(client, cfg)
	uiWrite := func(name string, msg string) {
		a.WriteText(name, msg)
	}
	uiUpdateContact := func(id string, name string) {
		a.UpdateContact(id, name)
	}
	if err := client.HandShake(name, uiWrite, uiUpdateContact); err != nil {
		return fmt.Errorf("error HandShake:%w", err)
	}
	a.WriteText("system", "CONNECTED")
	if err := a.Run(); err != nil {
		return fmt.Errorf("error running app:%w", err)
	}
	return nil
}
