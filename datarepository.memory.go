// datarepository.memory.go

package datarepository

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	nuts "github.com/vaudience/go-nuts"
)

type MemoryConfig struct {
	// Add any configuration options if needed
}

func (c MemoryConfig) GetConnectionString() string {
	return "memory://"
}

type MemoryIdentifier string

func (mi MemoryIdentifier) String() string {
	return string(mi)
}

type MemoryRepository struct {
	BaseRepository
	mu       sync.RWMutex
	data     map[string]interface{}
	locks    map[string]time.Time
	channels map[string][]chan interface{}
	expiries map[string]time.Time
}

func NewMemoryRepository(config Config) (DataRepository, error) {
	_, ok := config.(MemoryConfig)
	if !ok {
		return nil, fmt.Errorf("invalid config type for Memory repository")
	}

	repo := &MemoryRepository{
		data:     make(map[string]interface{}),
		locks:    make(map[string]time.Time),
		channels: make(map[string][]chan interface{}),
	}

	nuts.Interval(func() bool {
		repo.cleanupExpired()
		return true
	}, 1*time.Minute, false)

	return repo, nil
}

func (r *MemoryRepository) Create(ctx context.Context, identifier EntityIdentifier, value interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := identifier.String()
	if _, exists := r.data[key]; exists {
		return ErrAlreadyExists
	}
	r.data[key] = value
	return nil
}

func (r *MemoryRepository) Read(ctx context.Context, identifier EntityIdentifier, value interface{}) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := identifier.String()
	expiry, exists := r.expiries[key]
	if exists && time.Now().After(expiry) {
		delete(r.data, key)
		delete(r.expiries, key)
		return ErrNotFound
	}
	data, exists := r.data[key]
	if exists {
		// Assuming value is a pointer to the correct type
		*(value.(*interface{})) = data
		return nil
	}
	return ErrNotFound
}

func (r *MemoryRepository) Update(ctx context.Context, identifier EntityIdentifier, value interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := identifier.String()
	if _, exists := r.data[key]; !exists {
		return ErrNotFound
	}
	r.data[key] = value
	return nil
}

func (r *MemoryRepository) Upsert(ctx context.Context, identifier EntityIdentifier, value interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := identifier.String()
	r.data[key] = value
	return nil
}

func (r *MemoryRepository) Delete(ctx context.Context, identifier EntityIdentifier) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := identifier.String()
	if _, exists := r.data[key]; !exists {
		return ErrNotFound
	}
	delete(r.data, key)
	return nil
}

func (r *MemoryRepository) List(ctx context.Context, pattern EntityIdentifier) ([]EntityIdentifier, []interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	regex, err := regexp.Compile(strings.ReplaceAll(pattern.String(), "*", ".*"))
	if err != nil {
		return nil, nil, fmt.Errorf("%w: invalid pattern", ErrInvalidInput)
	}

	var results []interface{}
	var ids []EntityIdentifier
	for key := range r.data {
		if regex.MatchString(key) {
			ids = append(ids, MemoryIdentifier(key))
			entity := r.data[key]
			results = append(results, entity)
		}
	}
	return ids, results, nil
}

func (r *MemoryRepository) Search(ctx context.Context, query string, offset, limit int, sortBy, sortDir string) ([]EntityIdentifier, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if offset < 0 || limit < 0 {
		return nil, fmt.Errorf("%w: invalid offset or limit", ErrInvalidInput)
	}
	// This is a simple implementation. In a real-world scenario, you'd want to implement
	// a more sophisticated search algorithm.
	var result []EntityIdentifier
	for key, value := range r.data {
		if strings.Contains(fmt.Sprintf("%v", value), query) {
			result = append(result, MemoryIdentifier(key))
		}
	}

	// Sort results
	sort.Slice(result, func(i, j int) bool {
		if sortDir == "DESC" {
			i, j = j, i
		}
		return result[i].String() < result[j].String()
	})

	// Apply offset and limit
	if offset >= len(result) {
		return []EntityIdentifier{}, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func (r *MemoryRepository) AcquireLock(ctx context.Context, identifier EntityIdentifier, ttl time.Duration) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := identifier.String()
	if lockTime, exists := r.locks[key]; exists && time.Now().Before(lockTime) {
		return false, nil
	}
	r.locks[key] = time.Now().Add(ttl)
	return true, nil
}

func (r *MemoryRepository) ReleaseLock(ctx context.Context, identifier EntityIdentifier) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := identifier.String()
	if _, exists := r.locks[key]; !exists {
		return ErrNotFound
	}
	delete(r.locks, key)
	return nil
}

func (r *MemoryRepository) Publish(ctx context.Context, channel string, message interface{}) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if channels, exists := r.channels[channel]; exists {
		for _, ch := range channels {
			select {
			case ch <- message:
			default:
				// Channel is full, skip this subscriber
			}
		}
	}
	return nil
}

func (r *MemoryRepository) Subscribe(ctx context.Context, channel string) (chan interface{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ch := make(chan interface{}, 100) // Buffer size of 100
	r.channels[channel] = append(r.channels[channel], ch)

	go func() {
		<-ctx.Done()
		r.mu.Lock()
		defer r.mu.Unlock()
		for i, subCh := range r.channels[channel] {
			if subCh == ch {
				r.channels[channel] = append(r.channels[channel][:i], r.channels[channel][i+1:]...)
				close(ch)
				break
			}
		}
	}()

	return ch, nil
}

func (r *MemoryRepository) Ping(ctx context.Context) error {
	return nil // Always successful for in-memory repository
}

func (r *MemoryRepository) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, channels := range r.channels {
		for _, ch := range channels {
			close(ch)
		}
	}
	r.channels = make(map[string][]chan interface{})
	return nil
}

func (r *MemoryRepository) SetExpiration(ctx context.Context, identifier EntityIdentifier, expiration time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := identifier.String()
	if _, exists := r.data[key]; !exists {
		return ErrNotFound
	}

	if r.expiries == nil {
		r.expiries = make(map[string]time.Time)
	}
	r.expiries[key] = time.Now().Add(expiration)
	return nil
}

func (r *MemoryRepository) GetExpiration(ctx context.Context, identifier EntityIdentifier) (time.Duration, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := identifier.String()
	if expiry, exists := r.expiries[key]; exists {
		return time.Until(expiry), nil
	}
	return 0, ErrNotFound
}

func (r *MemoryRepository) AtomicIncrement(ctx context.Context, identifier EntityIdentifier) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := identifier.String()
	value, exists := r.data[key]
	if !exists {
		r.data[key] = int64(1)
		return 1, nil
	}

	switch v := value.(type) {
	case int64:
		v++
		r.data[key] = v
		return v, nil
	default:
		return 0, ErrInvalidInput
	}
}

// Add a method to clean up expired keys
func (r *MemoryRepository) cleanupExpired() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for key, expiry := range r.expiries {
		if now.After(expiry) {
			delete(r.data, key)
			delete(r.expiries, key)
		}
	}
}

func (r *MemoryRepository) initBaseRepository() {
	r.BaseRepository = BaseRepository{
		plugins: make(map[string]RepositoryPlugin),
	}
}
