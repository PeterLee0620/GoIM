package main

import (
	"fmt"
	"os"

	"github.com/DavidLee0620/GoIM/chat/api/frontends/client/app"
)

/*
	 SAMPLE CONFIG FILE : chat/zarf/config.json
		{
			"user": {
				"id": "<user_id>",
				"name": "<user_name>"
			},
			"contacts": [
				{
					"id": "20723",
					"name": "Kevin Enriquez"
				},
				{
					"id": "58365",
					"name": "Bill Kennedy"
				}
			]
		}
*/
const (
	url            = "ws://localhost:3000/connect"
	configFilePath = "chat/zarf/client"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("Error:%s\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := app.NewContacts(configFilePath)
	if err != nil {
		return fmt.Errorf("error config:%w", err)
	}
	id := cfg.My().ID
	name := cfg.My().Name
	client := app.New(id, url, cfg)
	defer client.Close()
	a := app.NewApp(client, cfg)

	if err := client.HandShake(name, a.WriteText, a.UpdateContact); err != nil {
		return fmt.Errorf("error HandShake:%w", err)
	}
	a.WriteText("system", "CONNECTED")
	if err := a.Run(); err != nil {
		return fmt.Errorf("error running app:%w", err)
	}
	return nil
}
