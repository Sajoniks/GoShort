package middleware

import (
	"github.com/gorilla/mux"
	"github.com/sajoniks/GoShort/internal/http-server/helper"
	"github.com/sajoniks/GoShort/internal/http-server/metrics"
	"net/http"
	"time"
)

func NewHttpMetrics(metrics *metrics.HttpMetrics) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		var f http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {

			spy := helper.SpyResponse(w)

			t1 := time.Now()
			next.ServeHTTP(spy, r)
			t2 := time.Since(t1)

			metrics.RecordHttp(spy.StatusCode, r, w, t2)
		}
		return f
	}
}
