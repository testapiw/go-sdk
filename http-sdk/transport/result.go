package transport

import (
	"time"

	http "github.com/testapiw/go-sdk/http-sdk/client"
	"github.com/testapiw/go-sdk/http-sdk/contract"
)

// Result is the outcome of a request together with the metrics collected
// during its lifecycle. The transport collects these metrics but does not
// write them anywhere — logging and metric persistence are the application's
// responsibility.
type Result struct {
	// Response is the HTTP response, nil when the request failed.
	Response *http.Response

	// Error is the terminal error, nil on success.
	Error error

	// Operation identifies the request that produced this result.
	Operation contract.Operation

	// StartedAt is when the request lifecycle began.
	StartedAt time.Time

	// FinishedAt is when the request lifecycle ended.
	FinishedAt time.Time

	// Attempts is the number of HTTP calls made (including retries).
	Attempts int

	// Retries is the number of retries performed.
	Retries int

	// Waited is the total time spent waiting (backoff, rate limit).
	Waited time.Duration
}

// Duration returns the total time the request lifecycle took.
func (r *Result) Duration() time.Duration {
	if r.FinishedAt.IsZero() {
		return time.Since(r.StartedAt)
	}
	return r.FinishedAt.Sub(r.StartedAt)
}

// StatusCode returns the HTTP status code, or 0 when there is no response.
func (r *Result) StatusCode() int {
	if r.Response == nil {
		return 0
	}
	return r.Response.StatusCode
}
