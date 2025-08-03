package dbfile

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

const (
	dbDirName     = "db"
	dbMsgsDirName = "msgs"
	dbFileName    = "data.json"
)

type User struct {
	ID       common.Address
	Name     string
	Messages []string
}

type DB struct {
	filePath  string
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
		filePath: filePath,
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
		msgs, err := readMsgsFromDisk(db.filePath, id)
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

	df, err := readDBFromDisk(db.filePath)
	if err != nil {
		return User{}, fmt.Errorf("config read: %w", err)
	}

	dfu := dataFileUser{
		ID:   id,
		Name: name,
	}

	df.Contacts = append(df.Contacts, dfu)

	flushDBToDisk(db.filePath, df)

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

	if err := flushMsgToDisk(db.filePath, id, msg); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}

// =============================================================================
// Disk access functionality.

type dataFileUser struct {
	ID   common.Address `json:"id"`
	Name string         `json:"name"`
}

type dataFile struct {
	MyAccount dataFileUser   `json:"my_account"`
	Contacts  []dataFileUser `json:"contacts"`
}

func newDB(filePath string, myAccountID common.Address) (dataFile, error) {
	os.MkdirAll(filepath.Join(filePath, dbDirName), os.ModePerm)
	os.MkdirAll(filepath.Join(filePath, dbDirName, dbMsgsDirName), os.ModePerm)

	var df dataFile

	_, err := os.Stat(filepath.Join(filePath, dbDirName, dbFileName))
	switch {
	case err != nil:
		df, err = createDBOnDisk(filePath, myAccountID)

	default:
		df, err = readDBFromDisk(filePath)
		if df.MyAccount.ID != myAccountID {
			return dataFile{}, fmt.Errorf("id mismatch: got: %s exp: %s", df.MyAccount.ID.Hex(), myAccountID.Hex())
		}
	}

	if err != nil {
		return dataFile{}, fmt.Errorf("config: %w", err)
	}

	return df, nil
}

func createDBOnDisk(filePath string, myAccountID common.Address) (dataFile, error) {
	fileName := filepath.Join(filePath, dbDirName, dbFileName)

	os.MkdirAll(filePath, os.ModePerm)

	f, err := os.Create(fileName)
	if err != nil {
		return dataFile{}, fmt.Errorf("config data file create: %w", err)
	}
	defer f.Close()

	df := dataFile{
		MyAccount: dataFileUser{
			ID:   myAccountID,
			Name: "Anonymous",
		},
		Contacts: []dataFileUser{
			{
				ID:   common.Address{},
				Name: "Sample Contact",
			},
		},
	}

	jsonDoc, err := json.MarshalIndent(df, "", "    ")
	if err != nil {
		return dataFile{}, fmt.Errorf("config data file marshal: %w", err)
	}

	if _, err := f.Write(jsonDoc); err != nil {
		return dataFile{}, fmt.Errorf("config data file write: %w", err)
	}

	return df, nil
}

func readDBFromDisk(filePath string) (dataFile, error) {
	fileName := filepath.Join(filePath, dbDirName, dbFileName)

	f, err := os.Open(fileName)
	if err != nil {
		return dataFile{}, fmt.Errorf("id data file open: %w", err)
	}
	defer f.Close()

	var df dataFile
	if err := json.NewDecoder(f).Decode(&df); err != nil {
		return dataFile{}, fmt.Errorf("config decode: %w", err)
	}

	return df, nil
}

func flushDBToDisk(filePath string, df dataFile) error {
	fileName := filepath.Join(filePath, dbDirName, dbFileName)

	f, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("config data file create: %w", err)
	}
	defer f.Close()

	jsonDF, err := json.MarshalIndent(df, "", "    ")
	if err != nil {
		return fmt.Errorf("config data file marshal: %w", err)
	}

	if _, err := f.Write(jsonDF); err != nil {
		return fmt.Errorf("config data file write: %w", err)
	}

	return nil
}

func readMsgsFromDisk(filePath string, id common.Address) ([]string, error) {
	fileName := filepath.Join(filePath, dbDirName, dbMsgsDirName, id.Hex()+".msg")

	f, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("message file open: %w", err)
	}
	defer f.Close()

	var msgs []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		s := scanner.Text()
		msgs = append(msgs, s)
	}

	return msgs, nil
}

func flushMsgToDisk(filePath string, id common.Address, msg string) error {
	fileName := filepath.Join(filePath, dbDirName, dbMsgsDirName, id.Hex()+".msg")

	var f *os.File

	_, err := os.Stat(fileName)
	switch {
	case err != nil:

		f, err = os.Create(fileName)
		if err != nil {
			return fmt.Errorf("message file create: %w", err)
		}

	default:
		f, err = os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("message file open: %w", err)
		}
	}

	defer f.Close()

	if _, err := f.WriteString(msg + "\n"); err != nil {
		return fmt.Errorf("message file write: %w", err)
	}

	return nil
}
