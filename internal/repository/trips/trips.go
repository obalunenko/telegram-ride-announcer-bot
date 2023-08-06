package trips

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/gofrs/uuid/v5"
)

// ErrNotFound is returned when a trip is not found.
var ErrNotFound = errors.New("trip not found")

type Repository interface {
	// CreateTrip creates a new trip.
	CreateTrip(ctx context.Context, name, date, description string, createdBy int64) (*Trip, error)
	// ListTrips returns all trips.
	ListTrips(ctx context.Context) ([]*Trip, error)
	// GetTripByID returns a trip by ID.
	GetTripByID(ctx context.Context, id uuid.UUID) (*Trip, error)
	// UpdateTrip updates a trip.
	UpdateTrip(ctx context.Context, id uuid.UUID, params UpdateTripParams) error
	// DeleteTrip deletes a trip.
	DeleteTrip(ctx context.Context, id uuid.UUID) error
}

type UpdateTripParams struct {
	Name        *string
	Date        *string
	Description *string
	Completed   *bool
}

type Trip struct {
	ID          uuid.UUID
	Name        string
	Date        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   time.Time
	CreatedBy   int64
	Completed   bool
}

type inMemoryRepository struct {
	mu sync.RWMutex

	trips map[uuid.UUID]*Trip
}

func (i *inMemoryRepository) DeleteTrip(ctx context.Context, id uuid.UUID) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	trip, ok := i.trips[id]
	if !ok {
		return ErrNotFound
	}

	if !trip.DeletedAt.IsZero() {
		return ErrNotFound
	}
	deletedAt := time.Now()

	trip.DeletedAt = deletedAt

	i.trips[id] = trip

	return nil
}

func (i *inMemoryRepository) UpdateTrip(ctx context.Context, id uuid.UUID, params UpdateTripParams) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	trip, ok := i.trips[id]
	if !ok {
		return ErrNotFound
	}

	if !trip.DeletedAt.IsZero() {
		return ErrNotFound
	}

	if params.Name != nil {
		trip.Name = *params.Name
	}

	if params.Date != nil {
		trip.Date = *params.Date
	}

	if params.Description != nil {
		trip.Description = *params.Description
	}

	if params.Completed != nil {
		trip.Completed = *params.Completed
	}

	trip.UpdatedAt = time.Now()

	i.trips[id] = trip

	return nil
}

func NewInMemory() Repository {
	return &inMemoryRepository{
		mu:    sync.RWMutex{},
		trips: make(map[uuid.UUID]*Trip),
	}
}

func (i *inMemoryRepository) CreateTrip(ctx context.Context, name, date, description string, createdBy int64) (*Trip, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	trip := &Trip{
		ID:          uuid.Must(uuid.NewV4()),
		Name:        name,
		Date:        date,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		DeletedAt:   time.Time{},
		CreatedBy:   createdBy,
		Completed:   false,
	}

	i.trips[trip.ID] = trip

	return trip, nil
}

func (i *inMemoryRepository) ListTrips(ctx context.Context) ([]*Trip, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	trips := make([]*Trip, 0, len(i.trips))

	for _, t := range i.trips {
		if t.DeletedAt.IsZero() {
			trips = append(trips, t)
		}
	}

	return trips, nil
}

func (i *inMemoryRepository) GetTripByID(ctx context.Context, id uuid.UUID) (*Trip, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	trip, ok := i.trips[id]
	if !ok {
		return nil, ErrNotFound
	}

	if !trip.DeletedAt.IsZero() {
		return nil, ErrNotFound
	}

	return trip, nil
}
