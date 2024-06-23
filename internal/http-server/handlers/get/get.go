package get

import (
	"errors"
	"github.com/gorilla/mux"
	"github.com/sajoniks/GoShort/internal/api/v1/event/urls"
	"github.com/sajoniks/GoShort/internal/api/v1/response"
	"github.com/sajoniks/GoShort/internal/http-server/helper"
	"github.com/sajoniks/GoShort/internal/http-server/middleware"
	"github.com/sajoniks/GoShort/internal/mq"
	"github.com/sajoniks/GoShort/internal/store/interface"
	"github.com/sajoniks/GoShort/internal/trace"
	"go.uber.org/zap"
	"net/http"
)

func NewGetUrlHandler(store urlstore.Store, kafka *mq.KafkaWriterWorker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := middleware.GetLogging(r.Context())
		vars := mux.Vars(r)
		alias := vars["alias"]

		if alias == "" {
			log.Error("empty alias")
			_ = helper.WriteProblemJson(w, response.ErrorMsg("empty alias"))
			return
		}

		var resp response.BaseResponse
		url, err := store.GetURL(alias)
		if err != nil {
			log.Error("get url error", zap.Error(trace.WrapError(err)))
			if errors.Is(err, urlstore.ErrUrlNotFound) {
				resp = response.ErrorMsg("requested url was not found")
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				resp = response.ErrorMsg("server error")
			}
			_ = helper.WriteProblemJson(w, &resp)
			return
		}

		kafka.AddJsonMessage(urls.NewAccessedEvent(url, alias))

		log.Info("access url", zap.String("url", url), zap.String("alias", alias))
		http.Redirect(w, r, url, http.StatusFound)
	})
}
