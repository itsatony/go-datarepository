# go-datarepository

`go-datarepository` is a flexible and extensible data repository interface for Go applications. It provides a common interface for various data storage solutions, allowing you to easily switch between different backends or use multiple storage systems in your application.

## Features

- Generic interface for common data operations (CRUD, List, Search)
- Support for complex identifiers with the `EntityIdentifier` interface
- Built-in support for locking mechanisms
- Publish-Subscribe operations
- Extensible design for easy addition of new storage backends
- Redis implementation included out-of-the-box with comprehensive key management and validation
- In-memory implementation for testing and prototyping
- Factory pattern for easy repository creation and registration

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

	// Create an in-memory repository
	memoryConfig := datarepository.MemoryConfig{}
	memoryRepo, err := datarepository.CreateDataRepository("memory", memoryConfig)
	if err != nil {
		log.Fatalf("Failed to create in-memory repository: %v", err)
	}
	defer memoryRepo.Close()

	// Use the repositories...
}
```

### Available Repository Types

You can check the available repository types using the `GetRegisteredRepositoryTypes` function:

```go
types := datarepository.GetRegisteredRepositoryTypes()
fmt.Println("Available repository types:", types)
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

After registration, you can create an instance of your new repository using:

```go
myConfig := MyNewConfig{} // Implement the Config interface
repo, err := datarepository.CreateDataRepository("mynew", myConfig)
```

## Key Features of the Redis Implementation

- Comprehensive key management and validation
- Support for complex identifiers using `RedisIdentifier`
- Automatic key assembly and parsing
- Robust error handling and input validation
- Efficient search result parsing

## Key Features of the In-Memory Implementation

- Thread-safe operations using sync.RWMutex
- Simple pattern matching for List operations
- Basic search functionality with sorting and pagination
- Efficient Publish-Subscribe mechanism

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.