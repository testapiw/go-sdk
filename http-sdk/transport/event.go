package transport

import (
	http "github.com/testapiw/go-sdk/http-sdk/client"
)

type EventType uint8

const (
	EventSuccess EventType = iota
	EventHTTPError
	EventNetworkError
)

type Event struct {
	Operation Operation

	Request  http.Request
	Response *http.Response
	Error    error

	State map[string]any
}

func (e *Event) Set(key string, value any) {
	if e.State == nil {
		e.State = make(map[string]any)
	}

	e.State[key] = value
}

func (e *Event) Get(key string) (any, bool) {
	if e.State == nil {
		return nil, false
	}

	v, ok := e.State[key]
	return v, ok
}