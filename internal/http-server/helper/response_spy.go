package helper

import "net/http"

type ResponseWriterSpy struct {
	http.ResponseWriter
	StatusCode  int
	ContentSize int
}

func SpyResponse(w http.ResponseWriter) *ResponseWriterSpy {
	switch w.(type) {
	case *ResponseWriterSpy:
		return w.(*ResponseWriterSpy)
	default:
		return &ResponseWriterSpy{ResponseWriter: w}
	}
}

func (r *ResponseWriterSpy) Header() http.Header {
	return r.ResponseWriter.Header()
}

func (r *ResponseWriterSpy) Write(bytes []byte) (int, error) {
	r.ContentSize = len(bytes)
	if r.StatusCode == 0 {
		r.StatusCode = http.StatusOK
	}
	return r.ResponseWriter.Write(bytes)
}

func (r *ResponseWriterSpy) WriteHeader(statusCode int) {
	r.StatusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
