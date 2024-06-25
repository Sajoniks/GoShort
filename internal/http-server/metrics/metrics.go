package metrics

import (
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sajoniks/GoShort/internal/http-server/metrics/interface"
	"net/http"
	"strconv"
	"time"
)

type noOpMetrics struct {
}

func NewNoOpMetrics() metricsinterface.HttpMetricsService {
	return &noOpMetrics{}
}

func (n noOpMetrics) RecordHttp(status int, r *http.Request, d time.Duration) {
}

type HttpMetrics struct {
	apiRequests *prometheus.CounterVec
	apiTimings  *prometheus.HistogramVec
}

func NewHttpMetrics(reg prometheus.Registerer) *HttpMetrics {
	m := HttpMetrics{}
	m.apiTimings = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "goshort",
		Subsystem: "api",
		Name:      "request_duration",
		Help:      "duration of api requests",
	}, []string{"path", "method", "status"})
	m.apiRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "goshort",
		Subsystem: "api",
		Name:      "request",
		Help:      "count of api requests",
	}, []string{"path", "method", "status", "content_type"})
	reg.MustRegister(m.apiRequests, m.apiTimings)
	return &m
}

func (m *HttpMetrics) RecordHttp(status int, r *http.Request, w http.ResponseWriter, d time.Duration) {
	route := mux.CurrentRoute(r)
	template, _ := route.GetPathTemplate()
	m.apiTimings.With(prometheus.Labels{
		"path":   template,
		"method": r.Method,
		"status": strconv.Itoa(status),
	}).Observe(d.Seconds())
	m.apiRequests.With(prometheus.Labels{
		"path":         template,
		"method":       r.Method,
		"status":       strconv.Itoa(status),
		"content_type": w.Header().Get("Content-Type"),
	}).Add(1)
}
