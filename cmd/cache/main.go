package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	resp "github.com/sajoniks/GoShort/internal/api/v1/response"
	"github.com/sajoniks/GoShort/internal/config"
	"github.com/sajoniks/GoShort/internal/http-server/middleware"
	"github.com/sajoniks/GoShort/internal/logging"
	"github.com/sajoniks/GoShort/internal/trace"
	"go.uber.org/zap"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
)

var (
	client *redis.Client
	cfg    *config.AppConfig
	logger *zap.Logger
)

func putCacheUrlAlias(w http.ResponseWriter, r *http.Request) {
	request := struct {
		Url   string `json:"url"`
		Alias string `json:"alias"`
	}{}
	defer r.Body.Close()

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		w.Header().Set("Content-Type", "application/problem+json")
		if errors.Is(err, io.EOF) {
			w.WriteHeader(http.StatusOK)
			response := resp.ErrorMsg("content empty")
			bs, _ := json.Marshal(&response)
			w.Write(bs)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			response := resp.ErrorMsg("content decode error")
			bs, _ := json.Marshal(&response)
			w.Write(bs)
		}

		logger.Error("request decode error", trace.AsZapError(err))
		return
	}

	request.Url = strings.TrimSpace(request.Url)
	request.Alias = strings.TrimSpace(request.Alias)

	if request.Url == "" || request.Alias == "" {
		w.Header().Set("Content-Type", "application/problem+json")
		response := resp.ErrorMsg("url or alias is empty")
		bs, _ := json.Marshal(&response)
		w.Write(bs)
		logger.Error("validation error", trace.AsZapError(errors.New(response.Error)))
		return
	}

	logger = logger.With(
		zap.String("url", request.Url),
		zap.String("alias", request.Alias),
	)

	err = client.Set(context.Background(), request.Alias, request.Url, time.Hour*24).Err()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			w.WriteHeader(http.StatusRequestTimeout)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		logger.Error("cache error", trace.AsZapError(err))
		return
	}

	logger.Info("cached url")
	w.WriteHeader(http.StatusOK)
}

func getCacheUrlAlias(w http.ResponseWriter, r *http.Request) {
	var alias string
	defer r.Body.Close()

	if varsAlias, ok := mux.Vars(r)["alias"]; !ok {
		logger.Error("segment not found", zap.String("segment", "alias"))
		return
	} else {
		alias = varsAlias
	}

	url, err := client.Get(context.Background(), alias).Result()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			w.WriteHeader(http.StatusRequestTimeout)
			return
		} else if errors.Is(err, redis.Nil) {
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		logger.Error("get from cache error", trace.AsZapError(err))
		return
	}

	logger.Info("cache hit", zap.String("alias", alias))

	response := struct {
		resp.BaseResponse
		Url string `json:"url"`
	}{}
	response.BaseResponse = resp.Ok()
	response.Url = url

	bs, err := json.Marshal(&response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error("response error", trace.AsZapError(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(bs)
}

func main() {
	var err error
	cfg = config.MustLoad()
	logger, err = logging.ConfigureLogger(config.GetEnvironment(), cfg)
	if err != nil {
		log.Fatalf("failed to configure logger: %v", err)
	}

	opt, err := redis.ParseURL(cfg.Database.ConnectionString)
	if err != nil {
		log.Fatalf("failed to start redis client: %v", err)
	}

	client = redis.NewClient(opt).WithTimeout(time.Second * 5)

	serverMux := mux.NewRouter()
	serverMux.Methods("GET").Path("/{alias}").HandlerFunc(getCacheUrlAlias)
	serverMux.Methods("POST").Path("/set").HandlerFunc(putCacheUrlAlias)
	serverMux.Use(
		middleware.NewRequestId(),
		middleware.NewLogging(logger),
		middleware.NewRecoverer(),
	)

	serv := http.Server{
		Addr:    cfg.Server.Host,
		Handler: serverMux,
	}

	go func() {
		const servName = "cache"
		log.Printf("%s: run at %s", servName, serv.Addr)
		if hostErr := serv.ListenAndServe(); hostErr != nil {
			log.Fatalf("%s: error listening: %v", servName, hostErr)
		}
	}()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	<-c
	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
	serv.Shutdown(ctx)

	log.Print("Shut down")
	os.Exit(0)
}
