# go-datarepository

`go-datarepository` is a flexible and extensible data repository interface for Go applications. It provides a common interface for various data storage solutions, allowing you to easily switch between different backends or use multiple storage systems in your application.

## Features

- Generic interface for common data operations (CRUD, List, Search)
- Support for complex identifiers with the `EntityIdentifier` interface
- Built-in support for locking mechanisms
- Publish-Subscribe operations
- Extensible design for easy addition of new storage backends
- Redis implementation included out-of-the-box with comprehensive key management and validation

## Installation

To use `go-datarepository` in your Go project, you can install it using `go get`:

```bash
go get github.com/itsatony/go-datarepository
```

## Usage

Here's a quick example of how to use the Redis implementation of the `DataRepository` interface:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/itsatony/go-datarepository"
)

func main() {
	config := datarepository.RedisConfig{
		Addrs:        []string{"localhost:6379"},
		Mode:         "single",
		KeyPrefix:    "myapp",
		KeySeparator: ":",
	}

	repo, err := datarepository.CreateDataRepository("redis", config)
	if err != nil {
		log.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// Create an item
	id := datarepository.RedisIdentifier{EntityPrefix: "user", ID: "123"}
	err = repo.Create(ctx, id, map[string]interface{}{"name": "John Doe", "age": 30})
	if err != nil {
		log.Printf("Failed to create item: %v", err)
	}

	// Read an item
	var user map[string]interface{}
	err = repo.Read(ctx, id, &user)
	if err != nil {
		log.Printf("Failed to read item: %v", err)
	} else {
		fmt.Printf("User: %v\n", user)
	}

	// List items
	pattern := datarepository.RedisIdentifier{EntityPrefix: "user", ID: "*"}
	identifiers, err := repo.List(ctx, pattern)
	if err != nil {
		log.Printf("Failed to list items: %v", err)
	} else {
		fmt.Printf("User IDs: %v\n", identifiers)
	}

	// Search
	results, err := repo.Search(ctx, "@name:John", 0, 10, "age", "ASC")
	if err != nil {
		log.Printf("Failed to search: %v", err)
	} else {
		fmt.Printf("Search results: %v\n", results)
	}

	// Acquire a lock
	locked, err := repo.AcquireLock(ctx, id, time.Second*30)
	if err != nil {
		log.Printf("Failed to acquire lock: %v", err)
	} else {
		fmt.Printf("Lock acquired: %v\n", locked)
	}

	// Release a lock
	err = repo.ReleaseLock(ctx, id)
	if err != nil {
		log.Printf("Failed to release lock: %v", err)
	}

	// Publish a message
	err = repo.Publish(ctx, "notifications", "Hello, World!")
	if err != nil {
		log.Printf("Failed to publish message: %v", err)
	}

	// Subscribe to a channel
	ch, err := repo.Subscribe(ctx, "notifications")
	if err != nil {
		log.Printf("Failed to subscribe: %v", err)
	} else {
		go func() {
			for msg := range ch {
				fmt.Println("Received:", msg)
			}
		}()
	}
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

## Key Features of the Redis Implementation

- Comprehensive key management and validation
- Support for complex identifiers using `RedisIdentifier`
- Automatic key assembly and parsing
- Robust error handling and input validation
- Efficient search result parsing

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
