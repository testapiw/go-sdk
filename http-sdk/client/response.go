package http

import stdhttp "net/http"

type Response struct {
	StatusCode int
	Header     stdhttp.Header
	Body       []byte
}

func (r Response) Success() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

func (r Response) HeaderValue(name string) string {
	if r.Header == nil {
		return ""
	}
	return r.Header.Get(name)
}