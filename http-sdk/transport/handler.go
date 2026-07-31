package transport

import "context"

type Handler interface {
	Handle(
		context.Context,
		*Event,
	) Decision
}