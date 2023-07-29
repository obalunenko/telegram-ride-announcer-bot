package users

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrAlreadyExists = errors.New("user already exists")
	ErrNotFound      = errors.New("user not found")
)

type Repository interface {
	Create(ctx context.Context, user *User) error
	GetBuID(ctx context.Context, id int64) (*User, error)
	List(ctx context.Context) ([]*User, error)
}

type User struct {
	ID        int64
	Username  string
	Firstname string
	Lastname  string
}

// inMemoryRepository is an in-memory repository for users.
type inMemoryRepository struct {
	sync.RWMutex

	users map[int64]*User
}

func NewInMemory() Repository {
	return &inMemoryRepository{
		RWMutex: sync.RWMutex{},
		users:   make(map[int64]*User),
	}
}

func (i *inMemoryRepository) Create(ctx context.Context, user *User) error {
	i.Lock()
	defer i.Unlock()

	if _, ok := i.users[user.ID]; ok {
		return ErrAlreadyExists
	}

	i.users[user.ID] = user

	return nil
}

func (i *inMemoryRepository) GetBuID(ctx context.Context, id int64) (*User, error) {
	i.RLock()
	defer i.RUnlock()

	u, ok := i.users[id]
	if !ok {
		return nil, ErrNotFound
	}

	return u, nil
}

func (i *inMemoryRepository) List(ctx context.Context) ([]*User, error) {
	i.RLock()
	defer i.RUnlock()

	users := make([]*User, 0, len(i.users))

	for _, u := range i.users {
		users = append(users, u)
	}

	return users, nil
}
