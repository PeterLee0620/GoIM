package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/DavidLee0620/GoIM/chat/foundation/signature"
	"github.com/ethereum/go-ethereum/crypto"
)

const filePath = "chat/zarf/client"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Private Key
	fileName := filepath.Join(filePath, "key.ecdsa")

	if err := generatePrivateKey(fileName); err != nil {
		return fmt.Errorf("generatePrivateKey: %w", err)
	}

	// -------------------------------------------------------------------------
	// Public ID

	privateKey, err := crypto.LoadECDSA(fileName)
	if err != nil {
		return fmt.Errorf("loadECDSA: %w", err)
	}

	fmt.Println("*** CLIENT SIDE ***")
	id := crypto.PubkeyToAddress(privateKey.PublicKey)
	fmt.Printf("ID: %s\n", id.String())

	// -------------------------------------------------------------------------
	// Sign Data

	data := struct {
		FromID string `json:"fromID"`
		ToID   string `json:"toID"`
		Msg    string `json:"msg"`
		Nonce  uint64 `json:"nonce"`
	}{
		FromID: id.String(),
		ToID:   "20723",
		Msg:    "Hello, Kevin!",
		Nonce:  1,
	}

	v, r, s, err := signature.Sign(data, privateKey)
	if err != nil {
		return fmt.Errorf("sign: %w", err)
	}

	fmt.Println("V:", v)
	fmt.Println("R:", r)
	fmt.Println("S:", s)

	// -------------------------------------------------------------------------

	id2, err := signature.FromAddress(data, v, r, s)
	if err != nil {
		return fmt.Errorf("from address: %w", err)
	}

	fmt.Println("\n*** CAP SIDE ***")
	fmt.Printf("ID : %s\n", id.String())
	fmt.Printf("ID2: %s\n", id2)

	return nil
}

func generatePrivateKey(fileName string) error {
	if _, err := os.Stat(fileName); err == nil {
		return nil
	}

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return fmt.Errorf("generateKey: %w", err)
	}

	if err := crypto.SaveECDSA(fileName, privateKey); err != nil {
		return fmt.Errorf("saveECDSA: %w", err)
	}

	return nil
}
