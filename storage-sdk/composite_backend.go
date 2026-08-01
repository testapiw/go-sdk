package storage

import (
	"context"
	"fmt"
	"log"
)

// CompositeBackend combines storage (File) and cache (Redis)
type CompositeBackend struct {
	primary Backend // Primary persistent storage (FileBackend)
	cache   Backend // Fast cache (RedisBackend)
	logger  func(format string, args ...any) // Logs cache errors
}

func NewCompositeBackend(primary Backend, cache Backend) *CompositeBackend {
	return &CompositeBackend{
		primary: primary,
		cache:   cache,
		logger:  log.Printf,
	}
}

// Get implements the Read-Through logic
func (c *CompositeBackend) Get(ctx context.Context, key string) ([]byte, error) {
	// 1. Try to get data from the cache (Redis)
	data, err := c.cache.Get(ctx, key)
	if err == nil {
		return data, nil
	}

	// 2. If the data is not in the cache or Redis is unavailable, read from the file (Primary)
	data, err = c.primary.Get(ctx, key)
	if err != nil {
		return nil, err // If not in the file either -> ErrNotFound is returned
	}

	// 3. Cache warm-up (Backpropagation): if read from the file, write back to Redis
	if err := c.cache.Set(ctx, key, data); err != nil {
		c.logger("storage: cache warm-up failed for key %q: %v", key, err)
	}

	return data, nil
}

// Set implements the Write-Through logic
func (c *CompositeBackend) Set(ctx context.Context, key string, value []byte) error {
	// 1. Mandatory write to the persistent storage (File)
	if err := c.primary.Set(ctx, key, value); err != nil {
		return fmt.Errorf("failed to write to primary storage: %w", err)
	}

	// 2. Update the fast cache (Redis).
	// A Redis error does not block the main write to the file, but it is logged.
	if err := c.cache.Set(ctx, key, value); err != nil {
		c.logger("storage: cache write failed for key %q: %v", key, err)
	}

	return nil
}

// Delete invalidates data on both levels
func (c *CompositeBackend) Delete(ctx context.Context, key string) error {
	// Delete from the cache (an error does not block deletion from the primary storage)
	if err := c.cache.Delete(ctx, key); err != nil {
		c.logger("storage: cache delete failed for key %q: %v", key, err)
	}

	// Delete from the primary storage
	return c.primary.Delete(ctx, key)
}

// noopBackend is a stub used when the cache (Redis) is not configured.
// All operations return ErrNotFound/success without performing any real actions.
type noopBackend struct{}

func (noopBackend) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, ErrNotFound
}

func (noopBackend) Set(_ context.Context, _ string, _ []byte) error {
	return nil
}

func (noopBackend) Delete(_ context.Context, _ string) error {
	return nil
}