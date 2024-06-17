package helper

import "net/http"

type ResponseWriterSpy struct {
	http.ResponseWriter
	StatusCode  int
	ContentSize int
}

func (r *ResponseWriterSpy) Header() http.Header {
	return r.ResponseWriter.Header()
}

func (r *ResponseWriterSpy) Write(bytes []byte) (int, error) {
	r.ContentSize = len(bytes)
	return r.ResponseWriter.Write(bytes)
}

func (r *ResponseWriterSpy) WriteHeader(statusCode int) {
	r.StatusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
