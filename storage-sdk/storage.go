package storage

import (
	"context"
	"encoding/json"
	"errors"
)

var (
	ErrNotFound = errors.New("storage: key not found")
)

// Serializer is responsible for encoding/decoding structures into bytes (JSON, MsgPack, etc.)
type Serializer interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
}

// JSONSerializer is the default serializer implementation
type JSONSerializer struct{}

func (s JSONSerializer) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (s JSONSerializer) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// Backend is the low-level contract for storage engines (works with bytes)
type Backend interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte) error
	Delete(ctx context.Context, key string) error
}

// Storage is a high-level generic client for working with domain structures
type Storage[T any] struct {
	backend    Backend
	serializer Serializer
}

// NewStorage creates a new SDK instance for a specific data type T
func NewStorage[T any](backend Backend, serializer Serializer) *Storage[T] {
	if serializer == nil {
		serializer = JSONSerializer{}
	}
	return &Storage[T]{
		backend:    backend,
		serializer: serializer,
	}
}

// Get retrieves data by key and automatically decodes it into a T structure
func (s *Storage[T]) Get(ctx context.Context, key string) (T, error) {
	var zero T

	raw, err := s.backend.Get(ctx, key)
	if err != nil {
		return zero, err
	}

	var result T
	if err := s.serializer.Unmarshal(raw, &result); err != nil {
		return zero, err
	}

	return result, nil
}

// Set serializes a T structure and stores it under the given key
func (s *Storage[T]) Set(ctx context.Context, key string, value T) error {
	raw, err := s.serializer.Marshal(value)
	if err != nil {
		return err
	}

	return s.backend.Set(ctx, key, raw)
}

// Delete removes data by key
func (s *Storage[T]) Delete(ctx context.Context, key string) error {
	return s.backend.Delete(ctx, key)
}