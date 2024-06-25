package main

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sajoniks/GoShort/internal/config"
	"github.com/sajoniks/GoShort/internal/http-server/handlers/get"
	"github.com/sajoniks/GoShort/internal/http-server/handlers/save"
	"github.com/sajoniks/GoShort/internal/http-server/metrics"
	"github.com/sajoniks/GoShort/internal/http-server/middleware"
	"github.com/sajoniks/GoShort/internal/mq"
	"github.com/sajoniks/GoShort/internal/store/sqlite"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"time"
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

func main() {
	env := config.GetEnvironment()
	cfg := config.MustLoad()
	logger, _ := configureLogger(env, cfg)

	store, err := sqlite.NewSqliteStore(cfg.Database.ConnectionString, sqlite.NewStoreMetrics(prometheus.DefaultRegisterer))
	if err != nil {
		logger.Panic("unable to load database", zap.Error(err))
	}

	kafka := mq.NewKafkaWriterWorker(&cfg.Messaging.Kafka.Writers[0], logger)
	httpMetrics := metrics.NewHttpMetrics(prometheus.DefaultRegisterer)

	servMux := mux.NewRouter()
	servMux.Use(
		middleware.NewHttpMetrics(httpMetrics),
		middleware.NewRequestId(),
		middleware.NewLogging(logger),
		middleware.NewRecoverer(),
	)

	servMux.Methods("POST").Path("/").Handler(save.NewSaveUrlHandler(cfg.Server.Host, store, kafka))
	servMux.Methods("GET").Path("/{alias}").Handler(get.NewGetUrlHandler(store, kafka))

	serv := &http.Server{
		Addr:    cfg.Server.Host,
		Handler: servMux,
	}

	metricsMux := mux.NewRouter()
	metricsMux.Methods("GET").Path(cfg.Metrics.Path).Handler(promhttp.Handler())

	metricsServ := &http.Server{
		Addr:    path.Join(cfg.Metrics.Host),
		Handler: metricsMux,
	}

	go func() {
		const servName = "http"
		log.Printf("%s: run on %s", servName, serv.Addr)
		if hostErr := serv.ListenAndServe(); hostErr != nil {
			log.Fatalf("%s: error listening: %v", servName, hostErr)
		}
		log.Printf("%s: shut down", servName)
	}()

	go func() {
		const servName = "metrics"
		log.Printf("%s: run on %s", servName, metricsServ.Addr)
		if hostErr := metricsServ.ListenAndServe(); hostErr != nil {
			log.Fatalf("%s: error listening: %v", servName, hostErr)
		}
		log.Printf("%s: shut down", servName)
	}()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	<-c

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		metricsServ.Shutdown(ctx)
	}()
	go func() {
		serv.Shutdown(ctx)
	}()
	<-ctx.Done()

	logger.Info("Shut down")
	os.Exit(0)
}
