package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type User struct {
	ID       common.Address
	Name     string
	Messages []string
}

type Contacts struct {
	me       User
	contacts map[common.Address]User
	mu       sync.RWMutex
	fileName string
	filePath string
}

const configFileName = "config.json"

func NewContacts(filePath string, id common.Address) (*Contacts, error) {
	os.Mkdir(filepath.Join(filePath, "contacts"), os.ModePerm)
	fileName := filepath.Join(filePath, configFileName)
	var doc document
	_, err := os.Stat(fileName)
	switch {
	case err != nil:
		doc, err = createConfig(fileName, id)
	default:
		doc, err = readConfig(fileName)
		if doc.User.ID != id {
			return nil, fmt.Errorf("id mismatch: %w", err)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("config file error: %w", err)
	}
	contacts := make(map[common.Address]User, len(doc.Contacts))
	for _, user := range doc.Contacts {
		contacts[user.ID] = User{
			ID:   user.ID,
			Name: user.Name,
		}
	}
	cfg := Contacts{
		me: User{
			ID:   doc.User.ID,
			Name: doc.User.Name,
		},
		contacts: contacts,
		fileName: fileName,
		filePath: filePath,
	}
	return &cfg, nil

}

func (c *Contacts) My() User {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.me
}
func (c *Contacts) Contacts() []User {
	c.mu.RLock()
	defer c.mu.RUnlock()
	users := make([]User, 0, len(c.contacts))
	for _, user := range c.contacts {
		users = append(users, user)
	}

	return users
}
func (c *Contacts) LookupContact(id common.Address) (User, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	u, exists := c.contacts[id]
	if !exists {
		return User{}, fmt.Errorf("contact not found")
	}
	return u, nil
}
func (c *Contacts) AddContact(id common.Address, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	doc, err := readConfig(c.fileName)
	if err != nil {
		return fmt.Errorf("addcontact readConfig:%w", err)
	}
	newDocUser := docUser{
		ID:   id,
		Name: name,
	}
	doc.Contacts = append(doc.Contacts, newDocUser)
	if err := writeConfig(c.fileName, doc); err != nil { // 检查写入错误
		return fmt.Errorf("addcontact writeConfig:%w", err)
	}

	// 更新内存中的 contacts map
	c.contacts[id] = User{
		Name: newDocUser.Name,
		ID:   newDocUser.ID,
	}
	return nil
}
func (c *Contacts) AddMessage(id common.Address, msg string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	u, exists := c.contacts[id]
	if !exists {
		return fmt.Errorf("contact not found")
	}
	u.Messages = append(u.Messages, msg)
	c.contacts[id] = u
	if err := c.writeMessage(id, msg); err != nil {
		return fmt.Errorf("addmessage writeMessage:%w", err)
	}
	return nil
}

// =======================================
func (c *Contacts) readMessage(id common.Address) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	u, exists := c.contacts[id]
	if !exists {
		return fmt.Errorf("contact not found")
	}
	if len(u.Messages) > 0 {
		return nil
	}
	fileName := filepath.Join(c.filePath, "contacts", id.Hex()+".msg")

	f, err := os.Open(fileName)
	if err != nil {
		return nil
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		//获取当前行文本。
		s := scanner.Text()
		u.Messages = append(u.Messages, s)
	}
	c.contacts[id] = u
	return nil
}

func (c *Contacts) writeMessage(id common.Address, msg string) error {
	var f *os.File
	fileName := filepath.Join(c.filePath, "contacts", id.Hex()+".msg")
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

// =======================================
type docUser struct {
	ID   common.Address `json:"id"`
	Name string         `json:"name"`
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

func createConfig(fileName string, id common.Address) (document, error) {
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
			ID:   id,
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
