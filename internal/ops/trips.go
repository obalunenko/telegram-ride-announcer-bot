package ops

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid/v5"
	log "github.com/obalunenko/logger"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/trips"
)

// CreateTripParams is a params for CreateTrip function.
type CreateTripParams struct {
	Name        string
	Date        string
	Description string
	CreatedBy   int64
}

// CreateTrip creates a new trip.
func CreateTrip(ctx context.Context, b backends, p CreateTripParams) (*models.Trip, error) {
	t, err := b.TripsRepository().CreateTrip(ctx, p.Name, p.Date, p.Description, p.CreatedBy)
	if err != nil {
		return nil, err
	}

	log.WithFields(ctx, log.Fields{
		"trip_id": t.ID,
	}).Debug("New trip created")

	return GetTrip(ctx, b, t.ID)
}

// GetTrip returns trip by ID.
func GetTrip(ctx context.Context, b backends, id uuid.UUID) (*models.Trip, error) {
	t, err := b.TripsRepository().GetTripByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get trip by ID: %w", err)
	}

	user, err := b.UsersRepository().GetBuID(ctx, t.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("get user by ID: %w", err)
	}

	return &models.Trip{
		ID:          t.ID,
		Name:        t.Name,
		Date:        t.Date,
		Description: t.Description,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		CreatedBy: &models.User{
			ID:        user.ID,
			Username:  user.Username,
			Firstname: user.Firstname,
			Lastname:  user.Lastname,
		},
	}, nil
}

// UpdateTripParams is a params for UpdateTrip function.
type UpdateTripParams struct {
	Name        *string
	Date        *string
	Description *string
	Completed   *bool
}

// UpdateTrip updates a trip.
func UpdateTrip(ctx context.Context, b backends, id uuid.UUID, p UpdateTripParams) (*models.Trip, error) {
	err := b.TripsRepository().UpdateTrip(ctx, id, trips.UpdateTripParams{
		Name:        p.Name,
		Date:        p.Date,
		Description: p.Description,
	})
	if err != nil {
		return nil, fmt.Errorf("update trip: %w", err)
	}

	log.WithFields(ctx, log.Fields{
		"trip_id": id,
	}).Debug("Trip updated")

	return GetTrip(ctx, b, id)
}

// DeleteTrip deletes a trip.
func DeleteTrip(ctx context.Context, b backends, id uuid.UUID) error {
	err := b.TripsRepository().DeleteTrip(ctx, id)
	if err != nil {
		return fmt.Errorf("delete trip: %w", err)
	}

	log.WithFields(ctx, log.Fields{
		"trip_id": id,
	}).Debug("Trip deleted")

	return nil
}

// ListTripsByUser return list of trips by user.
func ListTripsByUser(ctx context.Context, b backends, user *models.User) ([]*models.Trip, error) {
	list, err := b.TripsRepository().ListTripsByUser(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("list trips by user: %w", err)
	}

	result := make([]*models.Trip, 0, len(list))

	for _, t := range list {
		result = append(result, &models.Trip{
			ID:          t.ID,
			Name:        t.Name,
			Date:        t.Date,
			Description: t.Description,
			CreatedAt:   t.CreatedAt,
			UpdatedAt:   t.UpdatedAt,
			CreatedBy:   user,
		})
	}

	return result, nil
}
