package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const idFileName = "key.ecdsa"

func NewID(filePath string) (common.Address, error) {
	os.MkdirAll(filepath.Join(filePath, "id"), os.ModePerm)

	fileName := filepath.Join(filePath, "id", idFileName)

	var id common.Address

	_, err := os.Stat(fileName)
	switch {
	case err != nil:
		id, err = createKeyID(fileName)

	default:
		id, err = readKeyID(fileName)
	}

	if err != nil {
		return common.Address{}, fmt.Errorf("id: %w", err)
	}

	return id, nil
}

// =============================================================================

func createKeyID(fileName string) (common.Address, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return common.Address{}, fmt.Errorf("generateKey: %w", err)
	}

	if err := crypto.SaveECDSA(fileName, privateKey); err != nil {
		return common.Address{}, fmt.Errorf("saveECDSA: %w", err)
	}

	return crypto.PubkeyToAddress(privateKey.PublicKey), nil
}

func readKeyID(fileName string) (common.Address, error) {
	privateKey, err := crypto.LoadECDSA(fileName)
	if err != nil {
		return common.Address{}, fmt.Errorf("loadECDSA: %w", err)
	}

	return crypto.PubkeyToAddress(privateKey.PublicKey), nil
}
