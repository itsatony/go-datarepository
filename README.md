# go-datarepository

go-datarepository is a flexible and extensible data repository interface for Go applications. It provides a common interface for various data storage solutions, allowing you to easily switch between different backends or use multiple storage systems in your application.

## Version

v0.5.0

## Features

- Generic interface for common data operations (CRUD, List, Search)
- Support for complex identifiers with the `EntityIdentifier` interface
- Built-in support for expiration and atomic operations
- Publish-Subscribe operations
- Extensible design for easy addition of new storage backends
- Plugin system for database-specific optimizations
- Redis implementation included out-of-the-box with comprehensive key management and validation
- In-memory implementation for testing and prototyping
- Factory pattern for easy repository creation and registration
- Consistent error handling across different implementations

## Installation

To use go-datarepository in your Go project, you can install it using `go get`:

```bash
go get -u github.com/itsatony/go-datarepository
```

## Usage

To create a repository, use the `CreateDataRepository` function with the desired repository type and configuration:

```go
package main

import (
  "context"
  "fmt"
  "log"

  "github.com/itsatony/go-datarepository"
)

func main() {
  // Create a Redis repository
  redisConfig := datarepository.RedisConfig{
    ConnectionString: "single;appConnectionX;;;;;;0;localhost:6379",
    KeyPrefix: "superAppName",
    KeySeparator: ":",
  }

  redisRepo, err := datarepository.CreateDataRepository("redis", redisConfig)
  if err != nil {
    log.Fatalf("Failed to create Redis repository: %v", err)
  }
  defer redisRepo.Close()

  // Use the repository...
}
```

### New Methods

The `DataRepository` interface now includes the following new methods:

- `SetExpiration(ctx context.Context, identifier EntityIdentifier, expiration time.Duration) error`
- `GetExpiration(ctx context.Context, identifier EntityIdentifier) (time.Duration, error)`
- `AtomicIncrement(ctx context.Context, identifier EntityIdentifier) (int64, error)`

These methods provide support for setting and getting expiration times for keys, as well as performing atomic increment operations.

### Plugin System

go-datarepository now includes a plugin system for database-specific optimizations. You can create custom plugins by implementing the `RepositoryPlugin` interface:

```go
type RepositoryPlugin interface {
	Name() string
	Execute(ctx context.Context, command string, args ...interface{}) (interface{}, error)
}
```

To register a plugin with a repository:

```go
customPlugin := &MyCustomPlugin{}
err := repository.RegisterPlugin(customPlugin)
if err != nil {
    log.Fatalf("Failed to register plugin: %v", err)
}
```

To use a plugin:

```go
plugin, ok := repository.GetPlugin("MyCustomPlugin")
if ok {
    result, err := plugin.Execute(ctx, "CustomCommand", arg1, arg2)
    // Handle result and error
}
```

### In-Memory Implementation for Testing

go-datarepository includes an in-memory implementation that's well-suited for testing purposes. Instead of mocking a database, you can use this implementation in your tests for a more realistic behavior without external dependencies.

To use the in-memory implementation in your tests:

```go
package mypackage

import (
    "testing"
    "github.com/itsatony/go-datarepository"
)

func TestMyFunction(t *testing.T) {
    // Create an in-memory repository
    repo, err := datarepository.CreateDataRepository("memory", datarepository.MemoryConfig{})
    if err != nil {
        t.Fatalf("Failed to create in-memory repository: %v", err)
    }

    // Use the repository in your tests
    err = repo.Create(context.Background(), datarepository.SimpleIdentifier("testkey"), "testvalue")
    if err != nil {
        t.Errorf("Failed to create key: %v", err)
    }

    var value string
    err = repo.Read(context.Background(), datarepository.SimpleIdentifier("testkey"), &value)
    if err != nil {
        t.Errorf("Failed to read key: %v", err)
    }

    if value != "testvalue" {
        t.Errorf("Expected 'testvalue', got '%s'", value)
    }
}
```

Using the in-memory implementation for testing offers several advantages:

1. No need for external dependencies or database setup in your test environment.
2. Faster test execution compared to using a real database.
3. Consistent behavior across different test runs and environments.
4. Ability to test edge cases and error conditions easily.

Note that while the in-memory implementation is great for unit and integration tests, you should still perform end-to-end tests with your actual database to ensure full compatibility.

## Error Handling

The package provides consistent error types across different implementations:

- `ErrNotFound`: Returned when an entity is not found
- `ErrAlreadyExists`: Returned when trying to create an entity that already exists
- `ErrInvalidIdentifier`: Returned when an invalid identifier is provided
- `ErrInvalidInput`: Returned when invalid input is provided to a repository method
- `ErrOperationFailed`: Returned when a repository operation fails for a reason other than those above
- `ErrNotSupported`: Returned when an operation is not supported by the current repository implementation

You can use the provided helper functions to check for specific error types:

```go
err := repo.Create(ctx, identifier, value)
if datarepository.IsAlreadyExistsError(err) {
  // Handle already exists error
} else if datarepository.IsInvalidIdentifierError(err) {
  // Handle invalid identifier error
} else if err != nil {
  // Handle other errors
}
```

## Contributing

Contributions to the go-datarepository package are welcome! Please feel free to submit a Pull Request.

## License

no license

