package dbfile

import (
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type User struct {
	ID       common.Address
	Name     string
	Messages []string
}

type DB struct {
	myAccount User
	cache     map[common.Address]User
	mu        sync.RWMutex
}

func NewDB(filePath string, myAccountID common.Address) (*DB, error) {
	df, err := newDB(filePath, myAccountID)
	if err != nil {
		return nil, fmt.Errorf("newDB: %w", err)
	}

	cache := make(map[common.Address]User, len(df.Contacts))
	for _, user := range df.Contacts {
		cache[user.ID] = User{
			ID:   user.ID,
			Name: user.Name,
		}
	}

	db := DB{
		myAccount: User{
			ID:   df.MyAccount.ID,
			Name: df.MyAccount.Name,
		},
		cache: cache,
	}

	return &db, nil
}

func (c *DB) MyAccount() User {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.myAccount
}

func (c *DB) Contacts() []User {
	c.mu.RLock()
	defer c.mu.RUnlock()

	users := make([]User, 0, len(c.cache))
	for _, user := range c.cache {
		users = append(users, user)
	}

	return users
}

func (db *DB) QueryContactByID(id common.Address) (User, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	u, exists := db.cache[id]
	if !exists {
		return User{}, fmt.Errorf("contact not found")
	}

	if len(u.Messages) == 0 {
		msgs, err := readMsgsFromDisk(id)
		if err != nil {
			return User{}, fmt.Errorf("read messages: %w", err)
		}

		u.Messages = msgs
		db.cache[id] = u
	}

	return u, nil
}

func (db *DB) InsertContact(id common.Address, name string) (User, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	// -------------------------------------------------------------------------
	// Update in the in-memory cache of contacts.

	db.cache[id] = User{
		ID:   id,
		Name: name,
	}

	// -------------------------------------------------------------------------
	// Update the data.json file.

	df, err := readDBFromDisk()
	if err != nil {
		return User{}, fmt.Errorf("config read: %w", err)
	}

	dfu := dataFileUser{
		ID:   id,
		Name: name,
	}

	df.Contacts = append(df.Contacts, dfu)

	flushDBToDisk(df)

	// -------------------------------------------------------------------------
	// Return the new contact.

	u := User{
		ID:   id,
		Name: name,
	}

	return u, nil
}

func (db *DB) InsertMessage(id common.Address, msg string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	u, exists := db.cache[id]
	if !exists {
		return fmt.Errorf("contact not found")
	}

	u.Messages = append(u.Messages, msg)
	db.cache[id] = u

	if err := flushMsgToDisk(id, msg); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}
