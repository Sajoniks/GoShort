package main

import (
	"github.com/gorilla/mux"
	"github.com/sajoniks/GoShort/internal/config"
	"github.com/sajoniks/GoShort/internal/http-server/handlers/get"
	"github.com/sajoniks/GoShort/internal/http-server/handlers/save"
	"github.com/sajoniks/GoShort/internal/http-server/middleware"
	"github.com/sajoniks/GoShort/internal/store/sqlite"
	"go.uber.org/zap"
	"net/http"
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

	r := mux.NewRouter()
	r.Use(
		middleware.NewRequestId(),
		middleware.NewLogging(logger),
		middleware.NewRecoverer(),
	)

	r.Methods("POST").Path("/").Handler(save.NewSaveUrlHandler(store))
	r.Methods("GET").Path("/{alias}").Handler(get.NewGetUrlHandler(store))

	http.ListenAndServe(cfg.Server.Host, r)
}
