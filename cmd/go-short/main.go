package main

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/sajoniks/GoShort/internal/config"
	"github.com/sajoniks/GoShort/internal/http-server/handlers/get"
	"github.com/sajoniks/GoShort/internal/http-server/handlers/save"
	"github.com/sajoniks/GoShort/internal/http-server/middleware"
	"github.com/sajoniks/GoShort/internal/mq"
	"github.com/sajoniks/GoShort/internal/store/sqlite"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"os/signal"
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

	store, err := sqlite.NewSqliteStore(cfg.Database.ConnectionString)
	if err != nil {
		logger.Panic("unable to load database", zap.Error(err))
	}

	kafka := mq.NewKafkaWriterWorker(&cfg.Messaging.Kafka.Writers[0], logger)

	r := mux.NewRouter()
	r.Use(
		middleware.NewRequestId(),
		middleware.NewLogging(logger),
		middleware.NewRecoverer(),
	)

	r.Methods("POST").Path("/").Handler(save.NewSaveUrlHandler(store, kafka))
	r.Methods("GET").Path("/{alias}").Handler(get.NewGetUrlHandler(store, kafka))

	serv := &http.Server{
		Addr:    cfg.Server.Host,
		Handler: r,
	}

	go func() {
		if hostErr := serv.ListenAndServe(); hostErr != nil {
			log.Fatalf("error listening: %v", hostErr)
		}
	}()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	<-c

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serv.Shutdown(ctx)
	logger.Info("Shut down")

	os.Exit(0)
}
