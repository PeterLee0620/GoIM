package main

import (
	"fmt"
	"os"

	"github.com/DavidLee0620/GoIM/chat/api/frontends/client/app"
	dbfile "github.com/DavidLee0620/GoIM/chat/api/frontends/client/storage"
	"github.com/DavidLee0620/GoIM/chat/api/frontends/client/ui/tui"
)

const (
	url            = "ws://localhost:3000/connect"
	configFilePath = "chat/zarf/client"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	myAccountID, privateKey, err := app.NewID(configFilePath)
	if err != nil {
		return fmt.Errorf("id: %w", err)
	}

	db, err := dbfile.NewDB(configFilePath, myAccountID)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// -------------------------------------------------------------------------

	ui := tui.New(myAccountID, db.Contacts())

	// -------------------------------------------------------------------------

	app := app.NewApp(db, ui, myAccountID, privateKey, url)
	defer app.Close()

	ui.SetApp(app)

	// -------------------------------------------------------------------------

	if err := app.Handshake(db.MyAccount().Name); err != nil {
		return fmt.Errorf("handshake: %w", err)
	}

	if err := app.Run(); err != nil {
		return fmt.Errorf("run: %w", err)
	}

	return nil
}
