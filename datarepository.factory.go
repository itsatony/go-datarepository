// datarepository.factory.go

package datarepository

import (
	"fmt"
	"sync"
)

var (
	repositoryFactories = make(map[string]NewDataRepository)
	factoryMutex        sync.RWMutex
)

// RegisterDataRepository registers a new repository factory
func RegisterDataRepository(name string, factory NewDataRepository) {
	factoryMutex.Lock()
	defer factoryMutex.Unlock()
	repositoryFactories[name] = factory
}

// CreateDataRepository creates a new repository instance based on the provided name and config
func CreateDataRepository(name string, config Config) (DataRepository, error) {
	factoryMutex.RLock()
	factory, ok := repositoryFactories[name]
	factoryMutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown repository type: %s", name)
	}

	repo, err := factory(config)
	if err != nil {
		return nil, err
	}

	// Initialize BaseRepository for all repository types
	if baseRepo, ok := repo.(interface{ initBaseRepository() }); ok {
		baseRepo.initBaseRepository()
	}

	return repo, nil
}

// GetRegisteredRepositoryTypes returns a list of all registered repository types
func GetRegisteredRepositoryTypes() []string {
	factoryMutex.RLock()
	defer factoryMutex.RUnlock()

	types := make([]string, 0, len(repositoryFactories))
	for name := range repositoryFactories {
		types = append(types, name)
	}
	return types
}
