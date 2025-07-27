package app

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sync"
)

type User struct {
	ID   string
	Name string
}

type Config struct {
	user     User
	contacts map[string]User
	mu       sync.RWMutex
	fileName string
}

const configFileName = "config.json"

func NewConfig(filePath string) (*Config, error) {

	fileName := filepath.Join(filePath, configFileName)
	var doc document
	_, err := os.Stat(fileName)
	switch {
	case err != nil:
		doc, err = createConfig(fileName)
	default:
		doc, err = readConfig(fileName)
	}
	if err != nil {
		return nil, fmt.Errorf("config file error: %w", err)
	}
	contacts := make(map[string]User, len(doc.Contacts))
	for _, user := range doc.Contacts {
		contacts[user.ID] = User(user)
	}
	cfg := Config{
		user: User{
			ID:   doc.User.ID,
			Name: doc.User.Name,
		},
		contacts: contacts,
		fileName: fileName,
	}
	return &cfg, nil

}

func (c *Config) User() User {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.user
}
func (c *Config) Contacts() []User {
	c.mu.RLock()
	defer c.mu.RUnlock()
	users := make([]User, 0, len(c.contacts))
	for _, user := range c.contacts {
		users = append(users, user)
	}

	return users
}
func (c *Config) LookupContact(id string) (User, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	u, exists := c.contacts[id]
	if !exists {
		return User{}, fmt.Errorf("contact not found")
	}
	return u, nil
}
func (c *Config) AddContact(id string, name string) error {
	doc, err := readConfig(c.fileName)
	if err != nil {
		return fmt.Errorf("addcontact readConfig:%w", err)
	}
	newDocUser := docUser(User{ID: id, Name: name})
	doc.Contacts = append(doc.Contacts, newDocUser)
	if err := writeConfig(c.fileName, doc); err != nil { // 检查写入错误
		return fmt.Errorf("addcontact writeConfig:%w", err)
	}

	// 更新内存中的 contacts map
	c.contacts[id] = User(newDocUser)
	return nil
}

// =======================================

type docUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type document struct {
	User     docUser   `json:"user"`
	Contacts []docUser `json:"contacts"`
}

// =======================================
func readConfig(fileName string) (document, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return document{}, fmt.Errorf("id file open: %w", err)
	}
	defer f.Close()
	var doc document
	if err := json.NewDecoder(f).Decode(&doc); err != nil {
		return document{}, fmt.Errorf("id file docode: %w", err)
	}
	return doc, nil
}
func writeConfig(fileName string, doc document) error {
	f, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("id file Create: %w", err)
	}
	defer f.Close()
	jsonDoc, err := json.MarshalIndent(doc, "", "    ")
	if err != nil {
		return fmt.Errorf("config file MarshalIndent: %w", err)
	}
	if _, err := f.Write(jsonDoc); err != nil {
		return fmt.Errorf("config file write: %w", err)
	}
	return nil
}

func createConfig(fileName string) (document, error) {
	filePath := filepath.Dir(fileName)
	os.MkdirAll(filePath, os.ModePerm)
	f, err := os.Create(fileName)
	if err != nil {
		return document{}, fmt.Errorf("config file Create: %w", err)
	}
	defer f.Close()

	doc := document{
		User: docUser{
			Name: "Anonymous",
			ID:   fmt.Sprintf("%d", rand.IntN(99999)),
		},
		Contacts: []docUser{},
	}
	jsonDoc, err := json.MarshalIndent(doc, "", "    ")
	if err != nil {
		return document{}, fmt.Errorf("config file MarshalIndent: %w", err)
	}
	if _, err := f.Write(jsonDoc); err != nil {
		return document{}, fmt.Errorf("config file write: %w", err)
	}
	return doc, nil
}
