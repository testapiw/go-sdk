package base

import "time"

type Logger interface{ Log(LogEntry) }
type LogEntry struct {
	Endpoint, RequestID    string
	Duration               time.Duration
	StatusCode, RetryCount int
	Err                    error
}
type Metrics interface {
	Request(endpoint string)
	Error(endpoint string)
	Retry(endpoint string)
	Latency(endpoint string, duration time.Duration)
	Timeout(endpoint string)
	RateLimit(endpoint string)
}
type NopLogger struct{}

func (NopLogger) Log(LogEntry) {}

type NopMetrics struct{}

func (NopMetrics) Request(string)                {}
func (NopMetrics) Error(string)                  {}
func (NopMetrics) Retry(string)                  {}
func (NopMetrics) Latency(string, time.Duration) {}
func (NopMetrics) Timeout(string)                {}
func (NopMetrics) RateLimit(string)              {}
