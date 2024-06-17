package middleware

import (
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"net/http"
	"runtime/debug"
)

func NewRecoverer() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					if err == http.ErrAbortHandler {
						panic(err)
					}

					logger := r.Context().Value(LoggerCtxKey).(*zap.Logger)
					if logger == nil {
						debug.PrintStack()
					} else {
						logger.Error("handled panic",
							zap.Stack("panic stack"),
						)
					}

					w.WriteHeader(http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
