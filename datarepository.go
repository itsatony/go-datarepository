// datarepository.go

package datarepository

import (
	"context"
	"time"
)

// EntityIdentifier represents a unique identifier for an entity
type EntityIdentifier interface {
	// String returns a string representation of the identifier
	String() string
}

// SimpleIdentifier is a basic implementation of EntityIdentifier
type SimpleIdentifier string

func (si SimpleIdentifier) String() string {
	return string(si)
}

// DataRepository defines a generic interface for data storage operations
type DataRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, identifier EntityIdentifier, value interface{}) error
	Read(ctx context.Context, identifier EntityIdentifier, value interface{}) error
	Update(ctx context.Context, identifier EntityIdentifier, value interface{}) error
	Delete(ctx context.Context, identifier EntityIdentifier) error

	// Additional operations
	List(ctx context.Context, pattern EntityIdentifier) ([]EntityIdentifier, error)
	Search(ctx context.Context, query string, offset, limit int, sortBy, sortDir string) ([]EntityIdentifier, error)

	// Locking mechanisms
	AcquireLock(ctx context.Context, identifier EntityIdentifier, ttl time.Duration) (bool, error)
	ReleaseLock(ctx context.Context, identifier EntityIdentifier) error

	// Publish-Subscribe operations
	Publish(ctx context.Context, channel string, message interface{}) error
	Subscribe(ctx context.Context, channel string) (chan interface{}, error)

	// Misc
	Ping(ctx context.Context) error
	Close() error
}

// Config defines the configuration for a DataRepository
type Config interface {
	// GetConnectionString returns the connection string for the DataRepository
	GetConnectionString() string
}

// NewDataRepository creates a new DataRepository instance based on the provided config
type NewDataRepository func(config Config) (DataRepository, error)
