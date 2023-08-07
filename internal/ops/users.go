// Package ops provide operations for business logic.
package ops

import (
	"context"

	log "github.com/obalunenko/logger"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/users"
)

// GetUser returns user by ID.
func GetUser(ctx context.Context, b backends, userID int64) (*models.User, error) {
	// check is user exists
	user, err := b.UsersRepository().GetBuID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &models.User{
		ID:        user.ID,
		Username:  user.Username,
		Firstname: user.Firstname,
		Lastname:  user.Lastname,
	}, nil
}

// CreateUserParams is a params for CreateUser function.
type CreateUserParams struct {
	UserID    int64
	Username  string
	Firstname string
	Lastname  string
}

// CreateUser creates a new user.
func CreateUser(ctx context.Context, b backends, p CreateUserParams) (*models.User, error) {
	err := b.UsersRepository().Create(ctx, &users.User{
		ID:        p.UserID,
		Username:  p.Username,
		Firstname: p.Firstname,
		Lastname:  p.Lastname,
	})
	if err != nil {
		return nil, err
	}

	log.WithFields(ctx, log.Fields{
		"user_id": p.UserID,
	}).Debug("New user created")

	return GetUser(ctx, b, p.UserID)
}
