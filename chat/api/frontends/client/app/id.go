package app

import (
	"crypto/ecdsa"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const idFileName = "key.ecdsa"

func NewID(filePath string) (common.Address, *ecdsa.PrivateKey, error) {
	os.MkdirAll(filepath.Join(filePath, "id"), os.ModePerm)

	fileName := filepath.Join(filePath, "id", idFileName)

	var id common.Address
	var pk *ecdsa.PrivateKey
	_, err := os.Stat(fileName)
	switch {
	case err != nil:
		id, pk, err = createKeyID(fileName)

	default:
		id, pk, err = readKeyID(fileName)
	}

	if err != nil {
		return common.Address{}, nil, fmt.Errorf("id: %w", err)
	}

	return id, pk, nil
}

// =============================================================================

func createKeyID(fileName string) (common.Address, *ecdsa.PrivateKey, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("generateKey: %w", err)
	}

	if err := crypto.SaveECDSA(fileName, privateKey); err != nil {
		return common.Address{}, nil, fmt.Errorf("saveECDSA: %w", err)
	}

	return crypto.PubkeyToAddress(privateKey.PublicKey), privateKey, nil
}

func readKeyID(fileName string) (common.Address, *ecdsa.PrivateKey, error) {
	privateKey, err := crypto.LoadECDSA(fileName)
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("loadECDSA: %w", err)
	}

	return crypto.PubkeyToAddress(privateKey.PublicKey), privateKey, nil
}
