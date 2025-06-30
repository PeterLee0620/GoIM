package users

import (
	"context"
	"fmt"
	"sync"

	"github.com/DavidLee0620/GoIM/chat/app/sdk/chat"
	"github.com/DavidLee0620/GoIM/chat/foundation/logger"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Users struct {
	log   *logger.Logger
	users map[uuid.UUID]chat.User
	mu    sync.RWMutex
}

func New(log *logger.Logger) *Users {
	u := Users{
		log:   log,
		users: make(map[uuid.UUID]chat.User),
	}

	return &u
}

func (u *Users) AddUser(ctx context.Context, usr chat.User) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	if _, exists := u.users[usr.ID]; exists {
		return fmt.Errorf("user exists")
	}
	u.log.Info(ctx, "chat-adduser", "name", usr.Name, "id", usr.ID)

	u.users[usr.ID] = usr
	return nil
}
func (u *Users) RemoveUser(ctx context.Context, userID uuid.UUID) {
	u.mu.Lock()
	defer u.mu.Unlock()
	usr, exists := u.users[userID]
	if !exists {
		u.log.Info(ctx, "chat-removeuser", "userID", userID, "doesn't exisrs")
		return
	}
	u.log.Info(ctx, "chat-removeuser", "name", usr.Name, "id", usr.ID)
	delete(u.users, userID)
	usr.Conn.Close()
}

func (u *Users) Connections() map[uuid.UUID]*websocket.Conn {
	u.mu.RLock()
	defer u.mu.RUnlock()
	m := make(map[uuid.UUID]*websocket.Conn)
	for id, usr := range u.users {
		m[id] = usr.Conn
	}
	return m
}

func (u *Users) Retrieve(ctx context.Context, userID uuid.UUID) (chat.User, error) {
	u.mu.RLock()
	defer u.mu.RUnlock()
	usr, exists := u.users[userID]
	if !exists {
		return chat.User{}, chat.ErrNotExists
	}
	return usr, nil
}
