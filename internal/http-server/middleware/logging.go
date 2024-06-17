package middleware

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/sajoniks/GoShort/internal/http-server/helper"
	"go.uber.org/zap"
	"net/http"
	"time"
)

const (
	LoggerCtxKey = "loggerCtxKey"
)

func GetLogging(ctx context.Context) *zap.Logger {
	return ctx.Value(LoggerCtxKey).(*zap.Logger)
}

func NewLogging(logger *zap.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {

		handler := func(w http.ResponseWriter, r *http.Request) {
			child := logger.With(
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("user_agent", r.UserAgent()),
				zap.String("request_id", r.Header.Get("X-Request-ID")),
				zap.String("request_content_type", r.Header.Get("Content-Type")))

			r = r.WithContext(context.WithValue(r.Context(), LoggerCtxKey, child))
			spy := helper.ResponseWriterSpy{ResponseWriter: w}

			t1 := time.Now()
			defer func() {
				child.Info("request done",
					zap.Int("status_code", spy.StatusCode),
					zap.String("status", http.StatusText(spy.StatusCode)),
					zap.Int("response_content_size", spy.ContentSize),
					zap.String("time_taken", time.Since(t1).String()))
			}()

			next.ServeHTTP(&spy, r)
		}

		return http.HandlerFunc(handler)
	}
}
