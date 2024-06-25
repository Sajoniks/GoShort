package metricsinterface

import (
	"net/http"
	"time"
)

type HttpMetricsService interface {
	RecordHttp(status int, r *http.Request, d time.Duration)
}
