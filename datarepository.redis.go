// redis_repository.go

package datarepository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	DefaultKeyPrefix     = "app"
	DefaultKeySeparator  = ":"
	DefaultKeyPartsCount = 3 // prefix:entityPrefix:id
	MinKeyLength         = 5
	MaxKeyLength         = 256
	KeyPartLock          = "lock"
	KeyPartPubSubChannel = "channel"
)

var (
	ErrEmptyKeyPart           = errors.New("empty key part used but not allowed")
	ErrInvalidKeyFormat       = errors.New("invalid key format")
	ErrInvalidKeyLength       = errors.New("key length out of allowed range")
	ErrInvalidKeyPrefix       = errors.New("key does not start with the correct prefix")
	ErrInvalidKeySuffix       = errors.New("key does not have at least one part after prefix")
	ErrInvalidKeyChars        = errors.New("key contains invalid characters")
	ErrInvalidEntityPrefix    = errors.New("invalid entity prefix: must start with a letter and contain only letters, numbers, and underscores")
	ErrUnsupportedIdentifier  = errors.New("unsupported identifier type")
	ErrInvalidKeyPatternChars = errors.New("key-pattern contains invalid characters")

	validKeyRegex        = regexp.MustCompile(`^[a-zA-Z0-9_:.-]+$`)
	validKeyPatternRegex = regexp.MustCompile(`^[a-zA-Z0-9_:.\-\?\*]+$`)
	entityPrefixRegex    = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)
)

type RedisConfig struct {
	ConnectionString string
	KeyPrefix        string
	KeySeparator     string
}

type redisServerInfo struct {
	Mode             string
	Name             string
	MasterName       string
	SentinelUsername string
	SentinelPassword string
	Username         string
	Password         string
	DB               int
	Addrs            []string
}

func (c RedisConfig) GetConnectionString() string {
	return c.ConnectionString
}

func parseRedisServerInfoFromConfigString(redisConfigString string) (redisServerInfo, error) {
	rsi := redisServerInfo{}
	baseinfo := strings.Split(redisConfigString, ";")
	if len(baseinfo) < 9 {
		return rsi, fmt.Errorf("%w: invalid connection string format", ErrInvalidInput)
	}

	rsi.Mode = baseinfo[0]
	rsi.Name = baseinfo[1]
	rsi.MasterName = baseinfo[2]
	rsi.SentinelUsername = baseinfo[3]
	rsi.SentinelPassword = baseinfo[4]
	rsi.Username = baseinfo[5]
	rsi.Password = baseinfo[6]

	db, err := strconv.Atoi(baseinfo[7])
	if err != nil {
		rsi.DB = 0
	} else {
		rsi.DB = db
	}

	rsi.Addrs = strings.Split(baseinfo[8], ",")

	return rsi, nil
}

type RedisIdentifier struct {
	EntityPrefix string
	ID           string
}

func (ri RedisIdentifier) String() string {
	return strings.Join([]string{ri.EntityPrefix, ri.ID}, DefaultKeySeparator)
}

type RedisRepository struct {
	BaseRepository
	client    redis.UniversalClient
	prefix    string
	separator string
}

func (r *RedisRepository) initBaseRepository() {
	r.BaseRepository = BaseRepository{
		plugins: make(map[string]RepositoryPlugin),
	}
}

func NewRedisRepository(config Config) (DataRepository, error) {
	redisConfig, ok := config.(RedisConfig)
	if !ok {
		return nil, fmt.Errorf("%w: invalid config type for Redis repository", ErrInvalidInput)
	}
	if redisConfig.KeyPrefix == "" {
		redisConfig.KeyPrefix = DefaultKeyPrefix
	}
	if redisConfig.KeySeparator == "" {
		redisConfig.KeySeparator = DefaultKeySeparator
	}

	serverInfo, err := parseRedisServerInfoFromConfigString(redisConfig.ConnectionString)
	if err != nil {
		return nil, err
	}

	var client redis.UniversalClient

	switch serverInfo.Mode {
	case "single":
		client = redis.NewClient(&redis.Options{
			Addr:     serverInfo.Addrs[0],
			DB:       serverInfo.DB,
			Username: serverInfo.Username,
			Password: serverInfo.Password,
		})
	case "sentinel":
		client = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:       serverInfo.MasterName,
			SentinelAddrs:    serverInfo.Addrs,
			SentinelUsername: serverInfo.SentinelUsername,
			SentinelPassword: serverInfo.SentinelPassword,
			DB:               serverInfo.DB,
			Username:         serverInfo.Username,
			Password:         serverInfo.Password,
		})
	case "cluster":
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    serverInfo.Addrs,
			Username: serverInfo.Username,
			Password: serverInfo.Password,
		})
	default:
		return nil, fmt.Errorf("%w: unsupported Redis mode", ErrInvalidInput)
	}

	return &RedisRepository{
		client:    client,
		prefix:    redisConfig.KeyPrefix,
		separator: redisConfig.KeySeparator,
	}, nil
}

func (r *RedisRepository) validateKey(key string, allowPattern bool) error {
	if len(key) < MinKeyLength || len(key) > MaxKeyLength {
		return fmt.Errorf("%w: key length must be between %d and %d characters", ErrInvalidKeyLength, MinKeyLength, MaxKeyLength)
	}

	if allowPattern && !validKeyPatternRegex.MatchString(key) {
		return fmt.Errorf("%w: key-patterns must contain only alphanumeric characters, underscores, colons, dots, and hyphens and stars", ErrInvalidKeyPatternChars)
	} else if !allowPattern && !validKeyRegex.MatchString(key) {
		return fmt.Errorf("%w: key must contain only alphanumeric characters, underscores, colons, dots, and hyphens", ErrInvalidKeyChars)
	}

	if !strings.HasPrefix(key, r.prefix+r.separator) {
		return fmt.Errorf("%w: key must start with %s%s", ErrInvalidKeyPrefix, r.prefix, r.separator)
	}

	parts := strings.Split(key, r.separator)
	if len(parts) < 2 || parts[1] == "" {
		return fmt.Errorf("%w: key must have at least one non-empty part after the prefix", ErrInvalidKeySuffix)
	}

	for _, part := range parts {
		if part == "" {
			return ErrEmptyKeyPart
		}
	}

	return nil
}

func (r *RedisRepository) validateEntityPrefix(entityPrefix string) error {
	match := entityPrefixRegex.MatchString(entityPrefix)
	if !match {
		return ErrInvalidEntityPrefix
	}
	return nil
}

func (r *RedisRepository) createKey(parts ...string) (string, error) {
	allParts := append([]string{r.prefix}, parts...)
	key := strings.Join(allParts, r.separator)
	if err := r.validateKey(key, false); err != nil {
		return "", err
	}
	return key, nil
}

func (r *RedisRepository) createKeyPattern(parts ...string) (string, error) {
	allParts := append([]string{r.prefix}, parts...)
	key := strings.Join(allParts, r.separator)
	err := r.validateKey(key, true)
	if err != nil {
		return "", err
	}
	return key, nil
}

func (r *RedisRepository) parseKey(key string) ([]string, error) {
	if err := r.validateKey(key, false); err != nil {
		return nil, err
	}
	return strings.Split(key, r.separator)[1:], nil
}

func (r *RedisRepository) identifierToKey(identifier EntityIdentifier, allowPattern bool) (string, error) {
	switch id := identifier.(type) {
	case RedisIdentifier:
		err := r.validateEntityPrefix(id.EntityPrefix)
		// fmt.Printf("[]identifierToKey] ============== id EntityPrefix(%s) ID(%s) err(%v)\n", id.EntityPrefix, id.ID, err)
		if err != nil {
			return "", err
		}
		var key string
		if allowPattern {
			key, err = r.createKeyPattern(id.EntityPrefix, id.ID)
		} else {
			key, err = r.createKey(id.EntityPrefix, id.ID)
		}
		// fmt.Printf("[]identifierToKey] ============== allowPattern(%t) key(%s) (%v)\n", allowPattern, key, err)
		return key, err
	case SimpleIdentifier:
		return r.createKey(string(id))
	default:
		return "", ErrUnsupportedIdentifier
	}
}

func (r *RedisRepository) keyToIdentifier(key string) (EntityIdentifier, error) {
	parts, err := r.parseKey(key)
	if err != nil {
		return nil, err
	}
	if len(parts) >= 2 {
		return RedisIdentifier{EntityPrefix: parts[0], ID: parts[1]}, nil
	}
	return SimpleIdentifier(strings.Join(parts, r.separator)), nil
}

func (r *RedisRepository) Create(ctx context.Context, identifier EntityIdentifier, value interface{}) error {
	key, err := r.identifierToKey(identifier, false)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidIdentifier, err)
	}

	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists == 1 {
		return ErrAlreadyExists
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return r.client.Do(ctx, "JSON.SET", key, ".", string(data)).Err()
}

func (r *RedisRepository) Read(ctx context.Context, identifier EntityIdentifier, value interface{}) error {
	key, err := r.identifierToKey(identifier, false)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidIdentifier, err)
	}

	data, err := r.client.Do(ctx, "JSON.GET", key).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrNotFound
		}
		return err
	}

	return json.Unmarshal([]byte(data.(string)), value)
}

func (r *RedisRepository) Update(ctx context.Context, identifier EntityIdentifier, value interface{}) error {
	key, err := r.identifierToKey(identifier, false)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidIdentifier, err)
	}

	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		return ErrNotFound
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return r.client.Do(ctx, "JSON.SET", key, ".", string(data)).Err()
}

func (r *RedisRepository) Upsert(ctx context.Context, identifier EntityIdentifier, value interface{}) error {
	key, err := r.identifierToKey(identifier, false)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidIdentifier, err)
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return r.client.Do(ctx, "JSON.SET", key, ".", string(data)).Err()
}

func (r *RedisRepository) Delete(ctx context.Context, identifier EntityIdentifier) error {
	key, err := r.identifierToKey(identifier, false)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidIdentifier, err)
	}

	result, err := r.client.Del(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrOperationFailed, err)
	}
	if result == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *RedisRepository) List(ctx context.Context, pattern EntityIdentifier) ([]EntityIdentifier, []interface{}, error) {
	keyPattern, err := r.identifierToKey(pattern, true)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidIdentifier, err)
	}

	keys, err := r.client.Keys(ctx, keyPattern).Result()
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrOperationFailed, err)
	}

	identifiers := make([]EntityIdentifier, 0, len(keys))
	entities := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		err := r.validateKey(key, false)
		if err != nil {
			continue // Skip invalid keys
		}
		identifier, err := r.keyToIdentifier(key)
		if err != nil {
			continue // Skip keys that can't be converted to identifiers
		}
		// retrieve the value
		data, err := r.client.Do(ctx, "JSON.GET", key).Result()
		if err != nil {
			// return nil, nil, fmt.Errorf("%w: %v", ErrOperationFailed, err)
			// nuts.L.Debugf("Error getting value for key %s: %v", key, err)
			continue
		}
		entities = append(entities, data)
		identifiers = append(identifiers, identifier)
	}

	return identifiers, entities, nil
}

func (r *RedisRepository) Search(ctx context.Context, query string, offset, limit int, sortBy, sortDir string) ([]EntityIdentifier, error) {
	args := []interface{}{
		"FT.SEARCH", r.prefix, query,
		"LIMIT", offset, limit,
		"SORTBY", sortBy, sortDir,
	}
	res, err := r.client.Do(ctx, args...).Result()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOperationFailed, err)
	}

	array, ok := res.([]interface{})
	if !ok || len(array) < 1 {
		return nil, fmt.Errorf("unexpected search result format")
	}

	totalResults, ok := array[0].(int64)
	if !ok {
		return nil, fmt.Errorf("unexpected total results format")
	}

	if totalResults == 0 {
		return []EntityIdentifier{}, nil
	}

	identifiers := make([]EntityIdentifier, 0, totalResults)
	for i := 1; i < len(array); i += 2 {
		key, ok := array[i].(string)
		if !ok {
			continue // Skip invalid keys
		}
		if err := r.validateKey(key, false); err != nil {
			continue // Skip invalid keys
		}
		identifier, err := r.keyToIdentifier(key)
		if err != nil {
			continue // Skip keys that can't be converted to identifiers
		}
		identifiers = append(identifiers, identifier)
	}

	return identifiers, nil
}

func (r *RedisRepository) AcquireLock(ctx context.Context, identifier EntityIdentifier, ttl time.Duration) (bool, error) {
	key, err := r.identifierToKey(identifier, false)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrInvalidIdentifier, err)
	}
	lockKey := key + r.separator + KeyPartLock
	acquired, err := r.client.SetNX(ctx, lockKey, 1, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrOperationFailed, err)
	}
	return acquired, nil
}

func (r *RedisRepository) ReleaseLock(ctx context.Context, identifier EntityIdentifier) error {
	key, err := r.identifierToKey(identifier, false)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidIdentifier, err)
	}
	lockKey := key + r.separator + KeyPartLock
	result, err := r.client.Del(ctx, lockKey).Result()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrOperationFailed, err)
	}
	if result == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *RedisRepository) Publish(ctx context.Context, channel string, message interface{}) error {
	fullChannel := r.prefix + r.separator + KeyPartPubSubChannel + r.separator + channel
	return r.client.Publish(ctx, fullChannel, message).Err()
}

func (r *RedisRepository) Subscribe(ctx context.Context, channel string) (chan interface{}, error) {
	fullChannel := r.prefix + r.separator + KeyPartPubSubChannel + r.separator + channel
	pubsub := r.client.Subscribe(ctx, fullChannel)
	ch := make(chan interface{})

	go func() {
		defer close(ch)
		for msg := range pubsub.Channel() {
			ch <- msg.Payload
		}
	}()

	return ch, nil
}

func (r *RedisRepository) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisRepository) Close() error {
	return r.client.Close()
}

func (r *RedisRepository) SetExpiration(ctx context.Context, identifier EntityIdentifier, expiration time.Duration) error {
	key, err := r.identifierToKey(identifier, false)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidIdentifier, err)
	}
	return r.client.Expire(ctx, key, expiration).Err()
}

func (r *RedisRepository) GetExpiration(ctx context.Context, identifier EntityIdentifier) (time.Duration, error) {
	key, err := r.identifierToKey(identifier, false)
	if err != nil {
		return time.Duration(0), fmt.Errorf("%w: %v", ErrInvalidIdentifier, err)
	}
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if ttl < 0 {
		return 0, ErrNotFound
	}
	return ttl, nil
}

func (r *RedisRepository) AtomicIncrement(ctx context.Context, identifier EntityIdentifier) (int64, error) {
	key, err := r.identifierToKey(identifier, false)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidIdentifier, err)
	}
	return r.client.Incr(ctx, key).Result()
}
