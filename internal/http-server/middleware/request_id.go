package middleware

import (
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"net/http"
)

func NewRequestId() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Add("X-Request-ID", uuid.New().String())
			next.ServeHTTP(w, r)
		})
	}
}
