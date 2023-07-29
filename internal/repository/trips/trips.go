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
	// Create creates a new trip.
	Create(ctx context.Context, name, date, description string, createdBy int64) (*Trip, error)
	// List returns all trips.
	List(ctx context.Context) ([]*Trip, error)
	// GetByID returns a trip by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*Trip, error)
}

type Trip struct {
	ID          uuid.UUID
	Name        string
	Date        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedBy   int64
}

type inMemoryRepository struct {
	mu sync.RWMutex

	trips map[uuid.UUID]*Trip
}

func NewInMemory() Repository {
	return &inMemoryRepository{
		mu:    sync.RWMutex{},
		trips: make(map[uuid.UUID]*Trip),
	}
}

func (i *inMemoryRepository) Create(ctx context.Context, name, date, description string, createdBy int64) (*Trip, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	trip := &Trip{
		ID:          uuid.Must(uuid.NewV4()),
		Name:        name,
		Date:        date,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   createdBy,
	}

	i.trips[trip.ID] = trip

	return trip, nil
}

func (i *inMemoryRepository) List(ctx context.Context) ([]*Trip, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	trips := make([]*Trip, 0, len(i.trips))

	for _, t := range i.trips {
		trips = append(trips, t)
	}

	return trips, nil
}

func (i *inMemoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*Trip, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	trip, ok := i.trips[id]
	if !ok {
		return nil, ErrNotFound
	}

	return trip, nil
}
