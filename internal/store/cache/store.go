package cache

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/sajoniks/GoShort/internal/api/v1/response"
	urlstore "github.com/sajoniks/GoShort/internal/store/interface"
	"github.com/sajoniks/GoShort/internal/trace"
	"net/http"
	"net/url"
)

var (
	ErrTimeout            = errors.New("time out on request")
	ErrServerError        = errors.New("server error")
	ErrRequestError       = errors.New("error sending request")
	ErrRemoteStorageError = errors.New("remote storage error")
	ErrNoContent          = errors.New("no content")
)

type cacheStore struct {
	inner urlstore.Store
	addr  string
}

func (c *cacheStore) Close() {
	if v, ok := c.inner.(urlstore.CloseableStore); ok {
		v.Close()
	}
}

func (c *cacheStore) SaveURL(src, alias string) (string, error) {
	id, err := c.inner.SaveURL(src, alias)
	if err != nil {
		return "", trace.WrapError(ErrRemoteStorageError)
	}

	request := struct {
		Url   string `json:"url"`
		Alias string `json:"alias"`
	}{}
	request.Url = src
	request.Alias = alias
	requestUrl, err := url.JoinPath(c.addr, "set")
	if err != nil {
		return "", trace.WrapError(ErrRequestError)
	}
	buf := &bytes.Buffer{}
	_ = json.NewEncoder(buf).Encode(&request)
	resp, err := http.Post(requestUrl, "application/json", buf)
	if err != nil {
		return "", trace.WrapError(ErrRequestError)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusRequestTimeout {
			return "", trace.WrapError(ErrTimeout) // @todo retries?
		} else {
			return "", trace.WrapError(ErrRemoteStorageError)
		}
	}

	if resp.Header.Get("Content-Type") == "application/problem+json" {
		var cacheResponse response.BaseResponse
		decodeErr := json.NewDecoder(resp.Body).Decode(&cacheResponse)
		if decodeErr != nil {
			return "", trace.WrapError(ErrRemoteStorageError)
		} else {
			return "", errors.Join(trace.WrapError(ErrServerError), errors.New(cacheResponse.Error))
		}
	}
	return id, nil
}

func (c *cacheStore) GetURL(alias string) (string, error) {
	requestUrl, err := url.JoinPath(c.addr, alias)
	if err != nil {
		return "", trace.WrapError(ErrRequestError)
	}

	resp, err := http.Get(requestUrl)
	if err != nil {
		return "", trace.WrapError(ErrRequestError)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusRequestTimeout {
			return "", ErrTimeout // @todo retries?
		} else if resp.StatusCode == http.StatusNoContent {
			return c.inner.GetURL(alias) // no cached entry
		} else {
			return "", trace.WrapError(ErrRemoteStorageError)
		}
	}

	if resp.Header.Get("Content-Type") == "application/problem+json" {
		var cacheResponse response.BaseResponse
		decodeErr := json.NewDecoder(resp.Body).Decode(&cacheResponse)
		if decodeErr != nil {
			return "", trace.WrapError(ErrRemoteStorageError)
		} else {
			return "", errors.Join(trace.WrapError(ErrServerError), errors.New(cacheResponse.Error))
		}
	} else if resp.Header.Get("Content-Type") == "application/json" {
		var cacheResponse struct {
			response.BaseResponse
			Url string `json:"url"`
		}
		decodeErr := json.NewDecoder(resp.Body).Decode(&cacheResponse)
		if decodeErr != nil {
			return "", trace.WrapError(ErrRemoteStorageError)
		} else {
			return cacheResponse.Url, nil
		}
	} else {
		return "", trace.WrapError(ErrRemoteStorageError)
	}
}

func NewCachedStore(cacheAddr string, store urlstore.Store) (urlstore.CloseableStore, error) {
	if _, err := url.Parse(cacheAddr); err != nil {
		return nil, err
	}

	return &cacheStore{
		inner: store,
		addr:  cacheAddr,
	}, nil
}
