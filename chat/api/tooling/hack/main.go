package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/nats-io/nats.go"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := create(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(5)

	for range 5 {
		go func() {
			defer wg.Done()
			if err := recieve(); err != nil {
				log.Printf("Error receiving message: %v", err)
			}
		}()
	}

	publish()

	wg.Wait()

	return nil
}

func recieve() error {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		return fmt.Errorf("recieve: %w", err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		return fmt.Errorf("recieve: %w", err)
	}

	sub, err := js.SubscribeSync("cap")
	if err != nil {
		return fmt.Errorf("recieve: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), nats.DefaultTimeout)
	defer cancel()

	msg, err := sub.NextMsgWithContext(ctx)
	if err != nil {
		return fmt.Errorf("recieve: %w", err)
	}

	log.Printf("Received message: %s", string(msg.Data))
	return nil
}

func publish() error {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	defer nc.Close()

	// Create JetStream context
	js, err := nc.JetStream()
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	_, err = js.Publish("cap", []byte("Hello, JetStream!"))
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}

func create() error {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}

	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "cap",
		Subjects: []string{"cap"},
	})
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}

	return nil
}
