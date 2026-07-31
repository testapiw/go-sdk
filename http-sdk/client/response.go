package http

import stdhttp "net/http"

type Response struct {
	StatusCode int
	Header     stdhttp.Header
	Body       []byte
}

func (r *Response) Success() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

func (r *Response) IsClientError() bool {
	return r.StatusCode >= 400 && r.StatusCode < 500
}

func (r *Response) IsServerError() bool {
	return r.StatusCode >= 500
}