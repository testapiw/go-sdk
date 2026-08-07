package gateway

import "encoding/json"

// Request is the JSON document sent from the WP service to the collector.
//
// Action identifies the subscriber that should process the request.
// Payload is opaque to the transport and interpreted only by the subscriber.
type Request struct {
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload"`
}

// NewRequest builds a Request from a typed payload.
//
// The payload is marshalled to JSON and stored as raw bytes. Passing nil
// produces a request with an empty payload.
func NewRequest(action string, payload any) (Request, error) {
	if action == "" {
		return Request{}, ErrActionRequired
	}

	if payload == nil {
		return Request{Action: action}, nil
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return Request{}, err
	}

	return Request{
		Action:  action,
		Payload: raw,
	}, nil
}
