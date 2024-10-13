// datarepository.go

package datarepository

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotFound is returned when an entity is not found in the repository
	ErrNotFound = errors.New("entity not found")

	// ErrAlreadyExists is returned when trying to create an entity that already exists
	ErrAlreadyExists = errors.New("entity already exists")

	// ErrInvalidIdentifier is returned when an invalid identifier is provided
	ErrInvalidIdentifier = errors.New("invalid identifier")

	// ErrInvalidInput is returned when invalid input is provided to a repository method
	ErrInvalidInput = errors.New("invalid input")

	// ErrOperationFailed is returned when a repository operation fails for a reason other than those above
	ErrOperationFailed = errors.New("operation failed")

	// ErrNotSupported is returned when an operation is not supported by the repository
	ErrNotSupported = errors.New("operation not supported")
)

// DataRepository defines a generic interface for data storage operations
type DataRepository interface {
	// Create adds a new entity to the repository.
	// Returns ErrAlreadyExists if the entity already exists.
	// Returns ErrInvalidIdentifier if the identifier is invalid.
	Create(ctx context.Context, identifier EntityIdentifier, value interface{}) error

	// Read retrieves an entity from the repository.
	// Returns ErrNotFound if the entity does not exist.
	// Returns ErrInvalidIdentifier if the identifier is invalid.
	Read(ctx context.Context, identifier EntityIdentifier, value interface{}) error

	// Update modifies an existing entity in the repository.
	// Returns ErrNotFound if the entity does not exist.
	// Returns ErrInvalidIdentifier if the identifier is invalid.
	Update(ctx context.Context, identifier EntityIdentifier, value interface{}) error

	// Delete removes an entity from the repository.
	// Returns ErrNotFound if the entity does not exist.
	// Returns ErrInvalidIdentifier if the identifier is invalid.
	Delete(ctx context.Context, identifier EntityIdentifier) error

	// List returns entities matching the given pattern.
	// Returns ErrInvalidIdentifier if the pattern is invalid.
	List(ctx context.Context, pattern EntityIdentifier) ([]EntityIdentifier, error)

	// Search finds entities based on the given query.
	// Returns ErrInvalidInput if the search parameters are invalid.
	Search(ctx context.Context, query string, offset, limit int, sortBy, sortDir string) ([]EntityIdentifier, error)

	// AcquireLock attempts to acquire a lock for the given identifier.
	// Returns ErrInvalidIdentifier if the identifier is invalid.
	AcquireLock(ctx context.Context, identifier EntityIdentifier, ttl time.Duration) (bool, error)

	// ReleaseLock releases a previously acquired lock.
	// Returns ErrNotFound if the lock does not exist.
	// Returns ErrInvalidIdentifier if the identifier is invalid.
	ReleaseLock(ctx context.Context, identifier EntityIdentifier) error

	// Publish sends a message to the specified channel.
	Publish(ctx context.Context, channel string, message interface{}) error

	// Subscribe returns a channel that receives messages from the specified channel.
	Subscribe(ctx context.Context, channel string) (chan interface{}, error)

	// Ping checks the connection to the repository.
	// Returns ErrOperationFailed if the connection fails.
	Ping(ctx context.Context) error

	// Close releases any resources held by the repository.
	Close() error

	// SetExpiration sets the expiration time for the given identifier.
	SetExpiration(ctx context.Context, identifier EntityIdentifier, expiration time.Duration) error

	// GetExpiration returns the expiration time for the given identifier.
	GetExpiration(ctx context.Context, identifier EntityIdentifier) (time.Duration, error)

	// AtomicIncrement increments the value of the given identifier atomically.
	AtomicIncrement(ctx context.Context, identifier EntityIdentifier) (int64, error)

	// Plugin system
	RegisterPlugin(plugin RepositoryPlugin) error
	GetPlugin(name string) (RepositoryPlugin, bool)
}

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

// Config defines the configuration for a DataRepository
type Config interface {
	// GetConnectionString returns the connection string for the DataRepository
	GetConnectionString() string
}

// NewDataRepository creates a new DataRepository instance based on the provided config
type NewDataRepository func(config Config) (DataRepository, error)

// Init function to register all available repository types
func init() {
	// Register Redis repository
	RegisterDataRepository("redis", NewRedisRepository)

	// Register in-memory repository
	RegisterDataRepository("memory", NewMemoryRepository)

	// Add any additional repository registrations here
}

// IsNotFoundError checks if the given error is an ErrNotFound error
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsAlreadyExistsError checks if the given error is an ErrAlreadyExists error
func IsAlreadyExistsError(err error) bool {
	return errors.Is(err, ErrAlreadyExists)
}

// IsInvalidIdentifierError checks if the given error is an ErrInvalidIdentifier error
func IsInvalidIdentifierError(err error) bool {
	return errors.Is(err, ErrInvalidIdentifier)
}

// IsInvalidInputError checks if the given error is an ErrInvalidInput error
func IsInvalidInputError(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}

// IsOperationFailedError checks if the given error is an ErrOperationFailed error
func IsOperationFailedError(err error) bool {
	return errors.Is(err, ErrOperationFailed)
}

// RepositoryPlugin defines the interface for database-specific plugins
type RepositoryPlugin interface {
	Name() string
	Execute(ctx context.Context, command string, args ...interface{}) (interface{}, error)
}

// BaseRepository provides a basic implementation of the DataRepository interface
type BaseRepository struct {
	plugins map[string]RepositoryPlugin
}

// RegisterPlugin adds a new plugin to the repository
func (br *BaseRepository) RegisterPlugin(plugin RepositoryPlugin) error {
	if br.plugins == nil {
		br.plugins = make(map[string]RepositoryPlugin)
	}
	br.plugins[plugin.Name()] = plugin
	return nil
}

// GetPlugin returns the plugin with the given name
func (br *BaseRepository) GetPlugin(name string) (RepositoryPlugin, bool) {
	plugin, ok := br.plugins[name]
	return plugin, ok
}
