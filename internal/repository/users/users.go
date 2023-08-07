// Package users provides a repository for users.
package users

import (
	"context"
	"errors"
	"sync"
)

var (
	// ErrAlreadyExists is returned when user already exists.
	ErrAlreadyExists = errors.New("user already exists")
	// ErrNotFound is returned when user is not found.
	ErrNotFound = errors.New("user not found")
)

// Repository provides access to the user storage.
type Repository interface {
	// Create creates a new user.
	Create(ctx context.Context, user *User) error
	// GetBuID returns a user by ID.
	GetBuID(ctx context.Context, id int64) (*User, error)
	// List returns all users.
	List(ctx context.Context) ([]*User, error)
}

// User represents a user.
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

// NewInMemory creates a new in-memory repository for users.
func NewInMemory() Repository {
	return &inMemoryRepository{
		RWMutex: sync.RWMutex{},
		users:   make(map[int64]*User),
	}
}

func (i *inMemoryRepository) Create(_ context.Context, user *User) error {
	i.Lock()
	defer i.Unlock()

	if _, ok := i.users[user.ID]; ok {
		return ErrAlreadyExists
	}

	i.users[user.ID] = user

	return nil
}

func (i *inMemoryRepository) GetBuID(_ context.Context, id int64) (*User, error) {
	i.RLock()
	defer i.RUnlock()

	u, ok := i.users[id]
	if !ok {
		return nil, ErrNotFound
	}

	return u, nil
}

func (i *inMemoryRepository) List(_ context.Context) ([]*User, error) {
	i.RLock()
	defer i.RUnlock()

	users := make([]*User, 0, len(i.users))

	for _, u := range i.users {
		users = append(users, u)
	}

	return users, nil
}
