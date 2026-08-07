package gateway

// Response is returned by the collector after processing a request.
//
// Status is either:
//
//	"ok"
//	"error"
//
// Message is optional and normally contains an error description when
// Status == "error".
type Response struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// OK reports whether the response indicates a successful processing.
func (r Response) OK() bool {
	return r.Status == "ok"
}
