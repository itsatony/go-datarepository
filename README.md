# go-datarepository

`go-datarepository` is a flexible and extensible data repository interface for Go applications. It provides a common interface for various data storage solutions, allowing you to easily switch between different backends or use multiple storage systems in your application.

## Version

v0.3.0

## Features

- Generic interface for common data operations (CRUD, List, Search)
- Support for complex identifiers with the `EntityIdentifier` interface
- Built-in support for locking mechanisms
- Publish-Subscribe operations
- Extensible design for easy addition of new storage backends
- Redis implementation included out-of-the-box with comprehensive key management and validation
- In-memory implementation for testing and prototyping
- Factory pattern for easy repository creation and registration
- Consistent error handling across different implementations

## Installation

To use `go-datarepository` in your Go project, you can install it using `go get`:

```bash
go get github.com/itsatony/go-datarepository
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
		Addrs:        []string{"localhost:6379"},
		Mode:         "single",
		KeyPrefix:    "myapp",
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

### Error Handling

The package provides consistent error types across different implementations:

- `ErrNotFound`: Returned when an entity is not found
- `ErrAlreadyExists`: Returned when trying to create an entity that already exists
- `ErrInvalidIdentifier`: Returned when an invalid identifier is provided
- `ErrInvalidInput`: Returned when invalid input is provided to a repository method
- `ErrOperationFailed`: Returned when a repository operation fails for a reason other than those above

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

## Extending with New Backends

To add a new storage backend, implement the `DataRepository` interface and register it using the `RegisterDataRepository` function:

```go
type MyNewRepository struct {
	// ...
}

func NewMyNewRepository(config datarepository.Config) (datarepository.DataRepository, error) {
	// Initialize and return your new repository
}

func init() {
	datarepository.RegisterDataRepository("mynew", NewMyNewRepository)
}
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
