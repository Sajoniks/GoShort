package save

import (
	"bytes"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/sajoniks/GoShort/internal/api/v1/event/urls"
	resp "github.com/sajoniks/GoShort/internal/api/v1/response"
	"github.com/sajoniks/GoShort/internal/http-server/helper"
	"github.com/sajoniks/GoShort/internal/http-server/middleware"
	"github.com/sajoniks/GoShort/internal/mq"
	"github.com/sajoniks/GoShort/internal/store/interface"
	"github.com/sajoniks/GoShort/internal/trace"
	"go.uber.org/zap"
	"io"
	"net/http"
	"path"
	"regexp"
	"sync"
	"time"
)

// vars related to alias generation
var (
	aliasMutex         sync.Mutex
	aliasSha512        = sha512.New()
	aliasBuf           bytes.Buffer
	aliasBase64Encoder = base64.NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_").WithPadding(base64.NoPadding)
)

func generateAlias(url string) string {
	rnd := make([]byte, 8)
	if _, err := rand.Read(rnd); err != nil {
		return ""
	}

	aliasMutex.Lock()
	defer aliasMutex.Unlock()

	// url|timestamp|random bytes [8]
	// https://example.com|1234567890|jq2ef-=k
	fmt.Fprintf(&aliasBuf, "%s|%d|%s", url, time.Now().UTC().UnixNano(), rnd)

	_, err := aliasSha512.Write(aliasBuf.Bytes())
	aliasBuf.Reset()

	if err != nil {
		return ""
	}
	hsh := aliasSha512.Sum(nil)[:8]
	aliasSha512.Reset()

	return aliasBase64Encoder.EncodeToString(hsh)
}

type RequestSave struct {
	URL string `json:"url"`
}

type ResponseSave struct {
	resp.BaseResponse
	Alias string `json:"alias,omitempty"`
}

func NewSaveUrlHandler(
	baseHost string,
	store urlstore.Store,
	kafka mq.KafkaWriterWorkerInterface,
) http.HandlerFunc {
	urlRegex :=
		regexp.MustCompile("^(http:\\/\\/www\\.|https:\\/\\/www\\.|http:\\/\\/|https:\\/\\/|\\/|\\/\\/){1}[A-z0-9_-]*?[:]?[A-z0-9_-]*?[@]?[A-z0-9]+([\\-\\.]{1}[a-z0-9]+)*\\.[a-z]{2,5}(:[0-9]{1,5})?(\\/.*)?$")

	f := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := middleware.GetLogging(r.Context())

		var reqBody RequestSave

		if err := helper.DecodeJson(r.Body, &reqBody); err != nil {
			log.Error("error on decode json", zap.Error(trace.WrapError(err)))

			if errors.Is(err, io.EOF) {
				_ = helper.WriteProblemJson(w, &ResponseSave{
					BaseResponse: resp.ErrorMsg("empty request body"),
				})
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				_ = helper.WriteProblemJson(w, &ResponseSave{
					BaseResponse: resp.ErrorMsg("error decoding request content"),
				})
			}
			return
		}

		log = log.With(
			zap.String("source_url", reqBody.URL),
		)

		var reqResp ResponseSave
		if reqBody.URL == "" {
			reqResp.BaseResponse = resp.ErrorMsg("invalid url")
			log.Error("validation error", zap.String("error", reqResp.Error))

			_ = helper.WriteProblemJson(w, &reqResp)
			return
		}

		if !urlRegex.MatchString(reqBody.URL) {
			reqResp.BaseResponse = resp.ErrorMsg("invalid url")
			log.Error("validation error", zap.String("error", reqResp.Error))

			_ = helper.WriteProblemJson(w, &reqResp)
			return
		}

		alias := generateAlias(reqBody.URL)

		log = log.With(zap.String("alias", alias))

		id, err := store.SaveURL(reqBody.URL, alias)
		if err != nil {
			log.Error("save url error", zap.Error(trace.WrapError(err)))
			if errors.Is(err, urlstore.ErrUrlExists) {
				reqResp.BaseResponse = resp.ErrorMsg("url with alias is already added")
			} else if errors.Is(err, urlstore.ErrUrlEmpty) {
				reqResp.BaseResponse = resp.ErrorMsg("url is empty")
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				reqResp.BaseResponse = resp.ErrorMsg("server error")
			}

			_ = helper.WriteProblemJson(w, &reqResp)
			return
		}

		log.Info("added alias to url",
			zap.String("alias", alias),
			zap.String("url", reqBody.URL),
			zap.String("id", id),
		)

		kafka.AddJsonMessage(urls.NewAddedEvent(reqBody.URL, reqResp.Alias))

		reqResp.BaseResponse = resp.Ok()
		reqResp.Alias = path.Join(baseHost, alias)

		_ = helper.WriteJson(w, &reqResp)
	})

	return f
}
