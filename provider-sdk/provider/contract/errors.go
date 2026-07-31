package contract

import (
	"errors"
	"fmt"
)

var (
	ErrUnauthorized    = errors.New("provider: unauthorized")
	ErrRateLimit       = errors.New("provider: rate limit exceeded")
	ErrTimeout         = errors.New("provider: timeout")
	ErrUnavailable     = errors.New("provider: unavailable")
	ErrInvalidResponse = errors.New("provider: invalid response")
	ErrConfiguration   = errors.New("provider: invalid configuration")
	ErrNetwork         = errors.New("provider: network error")
	ErrInternal        = errors.New("provider: internal error")
)

type ProviderError struct {
	Kind      error
	Provider  string
	Operation string
	RequestID string
	Cause     error
}

func (e *ProviderError) Error() string {
	if e == nil {
		return "<nil>"
	}
	msg := fmt.Sprintf("%s %s: %v", e.Provider, e.Operation, e.Kind)
	if e.Cause != nil {
		msg += ": " + e.Cause.Error()
	}
	return msg
}
func (e *ProviderError) Unwrap() error {
	if e.Cause != nil {
		return errors.Join(e.Kind, e.Cause)
	}
	return e.Kind
}

func NewError(kind error, provider, operation, requestID string, cause error) error {
	return &ProviderError{Kind: kind, Provider: provider, Operation: operation, RequestID: requestID, Cause: cause}
}
