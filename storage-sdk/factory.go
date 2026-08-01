package storage

import (
	"fmt"
	"github.com/redis/go-redis/v9"
)

// Config holds the storage initialization parameters
type Config struct {
	FilePath  string
	RedisAddr string
	RedisPass string
	RedisDB   int
	// Optional pre-existing Redis client
	RedisClient *redis.Client
}

// Option defines a factory configuration function
type Option func(*Config)

// WithFilePath sets the directory for the file storage
func WithFilePath(path string) Option {
	return func(c *Config) {
		c.FilePath = path
	}
}

// WithRedisDB sets the Redis connection parameters
func WithRedisDB(addr, password string, db int) Option {
	return func(c *Config) {
		c.RedisAddr = addr
		c.RedisPass = password
		c.RedisDB = db
	}
}

// WithRedisClient allows passing an already existing *redis.Client
func WithRedisClient(client *redis.Client) Option {
	return func(c *Config) {
		c.RedisClient = client
	}
}

// Factory creates configured Storage instances
type Factory struct {
	backend Backend
}

// NewFactory is a factory method that hides the initialization of all backends
func NewFactory(opts ...Option) (*Factory, error) {
	// Default configuration values
	cfg := &Config{
		FilePath: "./data",
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// 1. Initialize the persistent backend (File)
	fileBackend, err := NewFileBackend(cfg.FilePath)
	if err != nil {
		return nil, fmt.Errorf("storage factory: failed to init file backend: %w", err)
	}

	// 2. Initialize the caching backend (Redis).
	// If Redis is not configured (no address and no client), a noop stub is used.
	var cacheBackend Backend
	switch {
	case cfg.RedisClient != nil:
		cacheBackend = NewRedisBackend(cfg.RedisClient)
	case cfg.RedisAddr != "":
		rdb := redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPass,
			DB:       cfg.RedisDB,
		})
		cacheBackend = NewRedisBackend(rdb)
	default:
		cacheBackend = noopBackend{}
	}

	// 3. Compose into a single Composite Backend
	compositeBackend := NewCompositeBackend(fileBackend, cacheBackend)

	return &Factory{
		backend: compositeBackend,
	}, nil
}

// CreateStorage initializes a typed storage for type T
func CreateStorage[T any](f *Factory) *Storage[T] {
	return NewStorage[T](f.backend, nil)
}