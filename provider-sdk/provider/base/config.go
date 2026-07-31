package base

// The base package intentionally defines no Logger or Metrics interfaces.
// The transport collects metrics into transport.Result, and the application
// decides how to log or persist them. Keeping observability out of the SDK
// core keeps the request path fast and dependency-free.