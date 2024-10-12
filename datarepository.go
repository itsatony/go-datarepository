// datarepository.go

package datarepository

import (
	"context"
	"time"
)

// DataRepository defines a generic interface for data storage operations
type DataRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, key string, value interface{}) error
	Read(ctx context.Context, key string, value interface{}) error
	Update(ctx context.Context, key string, value interface{}) error
	Delete(ctx context.Context, key string) error

	// Additional operations
	List(ctx context.Context, pattern string) ([]string, error)
	Search(ctx context.Context, query string, offset, limit int, sortBy, sortDir string) ([]string, error)

	// Locking mechanisms
	AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	ReleaseLock(ctx context.Context, key string) error

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
