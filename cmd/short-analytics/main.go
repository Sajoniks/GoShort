package main

import (
	"context"
	"encoding/json"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sajoniks/GoShort/internal/api/v1/event/urls"
	"github.com/sajoniks/GoShort/internal/config"
	"github.com/sajoniks/GoShort/internal/mq"
	"github.com/sajoniks/GoShort/internal/trace"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

var (
	metricCounterUrlGet = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "goshort",
		Subsystem: "metric",
		Name:      "total_url_requests",
		Help:      "number of processed url get requests",
	})

	metricHistUrlLength = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "goshort",
		Subsystem: "metric",
		Name:      "url_len",
		Help:      "length of the recorded url",
		Buckets:   []float64{8, 32, 40, 50, 64, 72, 88, 100, 128, 156, 172, 200},
	})
)

func configureLogger(env string, cfg *config.AppConfig) (*zap.Logger, error) {
	var logger *zap.Logger
	var err error

	fields := zap.Fields(
		zap.String("host", cfg.Server.Host),
		zap.String("env", config.GetEnvironment()),
	)
	switch env {
	case "dev":
		logger, err = zap.NewDevelopment(fields)
	default:
		logger, err = zap.NewProduction(fields)
	}

	return logger, err
}

func handleUrlEvent(eventValue []byte, logger *zap.Logger) error {
	eventType := struct {
		Type string `json:"type"`
	}{}
	err := json.Unmarshal(eventValue, &eventType)
	if err != nil {
		return trace.WrapError(err)
	}

	switch eventType.Type {
	case urls.EventTagUrlAdded:
		var ev urls.AddedEvent
		if err := json.Unmarshal(eventValue, &ev); err != nil {
			return trace.WrapError(err)
		}

		metricHistUrlLength.Observe(float64(len(ev.Source)))

		logger.Info("parsed event", zap.String("event_type", ev.Type))

	case urls.EventTagUrlAccessed:
		var ev urls.AccessedEvent
		if err := json.Unmarshal(eventValue, &ev); err != nil {
			return trace.WrapError(err)
		}

		metricCounterUrlGet.Add(1.0)

		logger.Info("parsed event", zap.String("event_type", ev.Type))
	}

	return nil
}

func main() {

	ctx, cancel := context.WithCancel(context.Background())

	cfg := config.MustLoad()
	logger, _ := configureLogger(config.GetEnvironment(), cfg)

	http.Handle("/metrics", promhttp.Handler())

	reader := mq.NewKafkaReaderWorker(&cfg.Messaging.Kafka.Readers[0], logger)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				logger.Info("shutting down message processing")
				return
			case bs := <-reader.C:
				err := handleUrlEvent(bs, logger.With(zap.Namespace("handle url")))
				if err != nil {
					logger.Error("failed to parse event", zap.Error(trace.WrapError(err)))
				} else {
					logger.Info("parsed event")
				}
			}
		}
	}()
	serv := http.Server{
		Addr:    cfg.Server.Host,
		Handler: nil,
	}
	go func() {
		if err := serv.ListenAndServe(); err != nil {
			logger.Fatal("failed to start prometheus http server", zap.Error(err))
		}
	}()

	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt)
	<-ch

	cancel()
	wg.Wait()
	reader.Shutdown()

	qctx, _ := context.WithTimeout(ctx, time.Second*10)
	serv.Shutdown(qctx)

	logger.Info("Shut down")

	os.Exit(0)
}
