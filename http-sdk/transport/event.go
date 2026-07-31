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
    Type     EventType

    Request  http.Request
    Response *http.Response
    Error    error
}