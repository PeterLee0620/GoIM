package app

import "github.com/ethereum/go-ethereum/common"

// User represents a user in the system.
type User struct {
	ID       common.Address
	Name     string
	Messages []string
}
