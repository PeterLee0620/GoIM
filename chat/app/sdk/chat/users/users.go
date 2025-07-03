package users

import (
	"context"
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
		return chat.ErrExists
	}
	u.log.Info(ctx, "chat-adduser", "name", usr.Name, "id", usr.ID)

	u.users[usr.ID] = usr

	return nil
}
func (u *Users) UpdateLastPing(ctx context.Context, usrID uuid.UUID) error {
	u.muUsers.Lock()
	defer u.muUsers.Unlock()
	usr, exists := u.users[usrID]
	if !exists {
		return chat.ErrNotExists

	}
	usr.LastPing = time.Now()
	u.users[usr.ID] = usr

	return nil
}
func (u *Users) UpdateLastPong(ctx context.Context, usrID uuid.UUID) (chat.User, error) {
	u.muUsers.Lock()
	defer u.muUsers.Unlock()
	usr, exists := u.users[usrID]
	if !exists {
		return chat.User{}, chat.ErrNotExists

	}
	usr.LastPong = time.Now()
	u.users[usr.ID] = usr

	return usr, nil
}
func (u *Users) RemoveUser(ctx context.Context, userID uuid.UUID) {
	u.muUsers.Lock()
	defer u.muUsers.Unlock()
	usr, exists := u.users[userID]
	if !exists {
		u.log.Info(ctx, "chat-removeuser", "userID", userID, "doesn't exisrs")
		return
	}
	delete(u.users, userID)
	u.log.Info(ctx, "chat-removeuser", "name", usr.Name, "id", usr.ID)

}

func (u *Users) Connections() map[uuid.UUID]chat.Connection {

	u.muUsers.RLock()
	defer u.muUsers.RUnlock()
	m := make(map[uuid.UUID]chat.Connection)
	for id, usr := range u.users {
		m[id] = chat.Connection{
			Conn:     usr.Conn,
			LastPong: usr.LastPong,
			LastPing: usr.LastPing,
		}

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
