package ops

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid/v5"
	log "github.com/obalunenko/logger"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/trips"
)

type CreateTripParams struct {
	Name        string
	Date        string
	Description string
	CreatedBy   int64
}

func CreateTrip(ctx context.Context, tripsRepo trips.Repository, p CreateTripParams) (*models.Trip, error) {
	t, err := tripsRepo.CreateTrip(ctx, p.Name, p.Date, p.Description, p.CreatedBy)
	if err != nil {
		return nil, err
	}

	log.WithFields(ctx, log.Fields{
		"trip_id": t.ID,
	}).Debug("New trip created")

	return GetTrip(ctx, tripsRepo, t.ID)
}

func GetTrip(ctx context.Context, tripsRepo trips.Repository, id uuid.UUID) (*models.Trip, error) {
	t, err := tripsRepo.GetTripByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &models.Trip{
		ID:          t.ID,
		Name:        t.Name,
		Date:        t.Date,
		Description: t.Description,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		CreatedBy:   t.CreatedBy,
	}, nil
}

type UpdateTripParams struct {
	Name        *string
	Date        *string
	Description *string
	Completed   *bool
}

func UpdateTrip(ctx context.Context, tripsRepo trips.Repository, id uuid.UUID, p UpdateTripParams) (*models.Trip, error) {
	err := tripsRepo.UpdateTrip(ctx, id, trips.UpdateTripParams{
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

	return GetTrip(ctx, tripsRepo, id)
}

func DeleteTrip(ctx context.Context, tripsRepo trips.Repository, id uuid.UUID) error {
	err := tripsRepo.DeleteTrip(ctx, id)
	if err != nil {
		return fmt.Errorf("delete trip: %w", err)
	}

	log.WithFields(ctx, log.Fields{
		"trip_id": id,
	}).Debug("Trip deleted")

	return nil
}
