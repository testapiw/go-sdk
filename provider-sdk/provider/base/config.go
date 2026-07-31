package base

import "time"

type Logger interface {
	Log(LogEntry)
}

type LogEntry struct {
	Endpoint   string
	RequestID  string
	Duration   time.Duration
	StatusCode int
	Err        error
}

type Metrics interface {
	Request(endpoint string)
	Response(endpoint string)
	Error(endpoint string)
	Latency(endpoint string, duration time.Duration)
}

type NopLogger struct{}

func (NopLogger) Log(LogEntry) {}

type NopMetrics struct{}

func (NopMetrics) Request(string)                  {}
func (NopMetrics) Response(string)                 {}
func (NopMetrics) Error(string)                    {}
func (NopMetrics) Latency(string, time.Duration)   {}