// Package ops provide operations for business logic.
package ops

import (
	"context"

	log "github.com/obalunenko/logger"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/users"
)

// GetUser returns user by ID.
func GetUser(ctx context.Context, usersRepo users.Repository, userID int64) (*models.User, error) {
	// check is user exists
	user, err := usersRepo.GetBuID(ctx, userID)
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

// CreateUser creates a new user.
func CreateUser(ctx context.Context, usersRepo users.Repository, userID int64, username, firstname, lastname string) (*models.User, error) {
	err := usersRepo.Create(ctx, &users.User{
		ID:        userID,
		Username:  username,
		Firstname: firstname,
		Lastname:  lastname,
	})
	if err != nil {
		return nil, err
	}

	log.WithFields(ctx, log.Fields{
		"user_id": userID,
	}).Debug("New user created")

	return GetUser(ctx, usersRepo, userID)
}
