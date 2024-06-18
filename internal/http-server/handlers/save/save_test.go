package save

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/sajoniks/GoShort/internal/http-server/middleware"
	"github.com/sajoniks/GoShort/internal/store/interface"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

var store urlstore.Store

type mockSaveStore struct {
	items map[string]string
}

func (m *mockSaveStore) SaveURL(src, alias string) (string, error) {
	if strings.TrimSpace(src) == "" {
		return "", urlstore.ErrUrlEmpty
	}
	if strings.TrimSpace(alias) == "" {
		return "", urlstore.ErrAliasEmpty
	}
	if _, ok := m.items[alias]; ok {
		return "", urlstore.ErrUrlExists
	}
	for _, v := range m.items {
		if v == src {
			return "", urlstore.ErrUrlExists
		}
	}
	m.items[alias] = src
	return "1", nil
}

func (m *mockSaveStore) GetURL(alias string) (string, error) {
	panic("not supported")
}

func TestMain(m *testing.M) {
	store = &mockSaveStore{items: map[string]string{
		"aaaa": "https://www.foo.bar",
	}}
	os.Exit(m.Run())
}

func TestSaveHandler(t *testing.T) {
	tt := []struct {
		name    string
		url     string
		respErr string
	}{
		{
			name: "success for https",
			url:  "https://www.example.com",
		},
		{
			name: "success for http",
			url:  "http://www.example.com",
		},
		{
			name:    "duplicate entry",
			url:     "https://www.foo.bar",
			respErr: "url with alias is already added",
		},
		{
			name:    "fail without scheme",
			url:     "www.example.com",
			respErr: "invalid url",
		},
		{
			name:    "fail for empty url",
			url:     "",
			respErr: "url is required",
		},
		{
			name:    "fail for blank",
			url:     "    ",
			respErr: "invalid url",
		},
		{
			name:    "fail for malformed",
			url:     "w.e",
			respErr: "invalid url",
		},
		{
			name:    "fail for malformed",
			url:     "https://",
			respErr: "invalid url",
		},
		{
			name:    "fail for malformed",
			url:     "http",
			respErr: "invalid url",
		},
		{
			name:    "fail for malformed",
			url:     "https",
			respErr: "invalid url",
		},
		{
			name:    "fail for malformed",
			url:     "://",
			respErr: "invalid url",
		},
		{
			name:    "fail for malformed",
			url:     "/",
			respErr: "invalid url",
		},
	}

	// create discarding logger
	logger := zap.NewNop()

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewSaveUrlHandler(store)
			b := &bytes.Buffer{}
			fmt.Fprintf(b, `{"url": "%s"}`, tc.url)

			req := httptest.NewRequest(http.MethodPost, "/", b)
			req = req.WithContext(context.WithValue(req.Context(), middleware.LoggerCtxKey, logger))
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			var resp ResponseSave
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))

			if tc.respErr == "" {
				require.Equal(t, http.StatusOK, rr.Code)
				require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
				require.Equal(t, "", resp.Error)
				require.Equal(t, true, resp.Ok)
				require.NotEqual(t, strings.TrimSpace(resp.Alias), "")
			} else {
				require.True(t, rr.Code == http.StatusInternalServerError || rr.Code == http.StatusOK)
				require.Equal(t, "application/problem+json", rr.Header().Get("Content-Type"))
				require.Equal(t, tc.respErr, resp.Error)
				require.Equal(t, false, resp.Ok)
			}
		})
	}
}
