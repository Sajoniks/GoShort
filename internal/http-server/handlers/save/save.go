package save

import (
	"bytes"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	resp "github.com/sajoniks/GoShort/internal/api/v1/response"
	"github.com/sajoniks/GoShort/internal/http-server/helper"
	"github.com/sajoniks/GoShort/internal/http-server/middleware"
	"github.com/sajoniks/GoShort/internal/store/interface"
	"github.com/sajoniks/GoShort/internal/trace"
	"go.uber.org/zap"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"
)

type RequestSave struct {
	URL string `json:"url"`
}

type ResponseSave struct {
	resp.BaseResponse
	Alias string `json:"alias,omitempty"`
}

var mx sync.Mutex

func generateAlias(url string) string {
	b := bytes.Buffer{}
	rnd := make([]byte, 8)
	if _, err := rand.Read(rnd); err != nil {
		return ""
	}
	mx.Lock()
	fmt.Fprintf(&b, "%s|%d|%s", url, time.Now().UTC().UnixNano(), rnd)
	mx.Unlock()

	sha := sha512.New()
	_, err := sha.Write(b.Bytes())
	if err != nil {
		return ""
	}
	enc := base64.NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_").WithPadding(base64.NoPadding)
	return enc.EncodeToString(sha.Sum(nil)[:8])
}

func NewSaveUrlHandler(store urlstore.Store) http.HandlerFunc {
	urlRegex :=
		regexp.MustCompile("^(http:\\/\\/www\\.|https:\\/\\/www\\.|http:\\/\\/|https:\\/\\/|\\/|\\/\\/){1}[A-z0-9_-]*?[:]?[A-z0-9_-]*?[@]?[A-z0-9]+([\\-\\.]{1}[a-z0-9]+)*\\.[a-z]{2,5}(:[0-9]{1,5})?(\\/.*)?$")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := middleware.GetLogging(r.Context())

		var reqBody RequestSave

		if err := helper.DecodeJson(r.Body, &reqBody); err != nil {
			log.Error("error on decode json", zap.Error(trace.WrapError(err)))

			w.WriteHeader(http.StatusInternalServerError)

			if errors.Is(err, io.EOF) {
				w.WriteHeader(http.StatusInternalServerError)
				_ = helper.WriteProblemJson(w, &ResponseSave{
					BaseResponse: resp.ErrorMsg("empty request body"),
				})
			} else {
				_ = helper.WriteProblemJson(w, &ResponseSave{
					BaseResponse: resp.ErrorMsg("error decoding request content"),
				})
			}
			return
		}

		var reqResp ResponseSave
		if reqBody.URL == "" {
			reqResp.BaseResponse = resp.ErrorMsg("url is required")
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

		reqResp.BaseResponse = resp.Ok()
		reqResp.Alias = alias

		_ = helper.WriteJson(w, &reqResp)
	})
}
