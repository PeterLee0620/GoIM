package users

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/DavidLee0620/GoIM/chat/app/sdk/chat"
	"github.com/DavidLee0620/GoIM/chat/foundation/logger"
	"github.com/google/uuid"
)

type Users struct {
	log     *logger.Logger
	users   map[uuid.UUID]chat.User
	muUsers sync.RWMutex
}

func New(log *logger.Logger) *Users {
	u := Users{
		log:   log,
		users: make(map[uuid.UUID]chat.User),
	}

	return &u
}

func (u *Users) AddUser(ctx context.Context, usr chat.User) error {
	u.muUsers.Lock()
	defer u.muUsers.Unlock()
	if _, exists := u.users[usr.ID]; exists {
		return fmt.Errorf("user exists")
	}
	u.log.Info(ctx, "chat-adduser", "name", usr.Name, "id", usr.ID)

	u.users[usr.ID] = usr

	h := func(appData string) error {
		u.muUsers.Lock()
		defer func() {
			u.muUsers.Unlock()
			u.log.Info(ctx, "pong-handler", "name", usr.Name, "id", usr.ID)
		}()
		usr, exists := u.users[usr.ID]
		if !exists {
			u.log.Info(ctx, "pong handler", "name", usr.Name, "id", usr.ID, "status", "dose not exists")
			return nil
		}
		usr.LastPong = time.Now()
		u.users[usr.ID] = usr
		return nil
	}
	usr.Conn.SetPongHandler(h)
	return nil
}
func (u *Users) RemoveUser(ctx context.Context, userID uuid.UUID) {
	u.muUsers.Lock()
	defer u.muUsers.Unlock()
	usr, exists := u.users[userID]
	if !exists {
		u.log.Info(ctx, "chat-removeuser", "userID", userID, "doesn't exisrs")
		return
	}
	u.log.Info(ctx, "chat-removeuser", "name", usr.Name, "id", usr.ID)
	delete(u.users, userID)
	usr.Conn.Close()
}

func (u *Users) Connections() map[uuid.UUID]chat.Connection {
	const maxWait = 5 * time.Second
	u.muUsers.RLock()
	defer u.muUsers.RUnlock()
	m := make(map[uuid.UUID]chat.Connection)
	for id, usr := range u.users {
		c := chat.Connection{
			Conn: usr.Conn,
		}

		if time.Since(usr.LastPong) <= maxWait {
			c.Valid = true
		}

		m[id] = c

	}
	return m
}

func (u *Users) Retrieve(ctx context.Context, userID uuid.UUID) (chat.User, error) {
	u.muUsers.RLock()
	defer u.muUsers.RUnlock()
	usr, exists := u.users[userID]
	if !exists {
		return chat.User{}, chat.ErrNotExists
	}
	return usr, nil
}
