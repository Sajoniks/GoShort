package get

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/sajoniks/GoShort/internal/api/v1/response"
	"github.com/sajoniks/GoShort/internal/http-server/middleware"
	"github.com/sajoniks/GoShort/internal/store/interface"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var store urlstore.Store
var router *mux.Router

type mockGetStore struct {
	items map[string]string
}

func (m *mockGetStore) SaveURL(src, alias string) (string, error) {
	panic("not supported")
}

func (m *mockGetStore) GetURL(alias string) (string, error) {
	if url, ok := m.items[alias]; ok {
		return url, nil
	} else {
		return "", urlstore.ErrUrlNotFound
	}
}

func TestMain(m *testing.M) {
	store = &mockGetStore{
		items: map[string]string{
			"aaaa": "https://www.example.com",
			"bbbb": "http://www.example.com",
		},
	}

	router = mux.NewRouter()
	router.Handle("/{alias}", NewGetUrlHandler(store))

	os.Exit(m.Run())
}

func TestGetHandler(t *testing.T) {
	tt := []struct {
		name    string
		alias   string
		respErr string
	}{
		{
			name:  "success",
			alias: "aaaa",
		},
		{
			name:  "success2",
			alias: "bbbb",
		},
		{
			name:    "failure",
			alias:   "cccc",
			respErr: "requested url was not found",
		},
		{
			name:    "failure",
			alias:   "",
			respErr: "empty alias",
		},
	}

	logger := zap.NewNop()

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			b := &bytes.Buffer{}

			req := httptest.NewRequest(http.MethodGet, "/"+tc.alias, b)
			req = req.WithContext(context.WithValue(req.Context(), middleware.LoggerCtxKey, logger))

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if tc.respErr == "" {
				require.Equal(t, http.StatusFound, rr.Code)
			} else {

				require.True(t, rr.Code == http.StatusOK || rr.Code == http.StatusInternalServerError || rr.Code == http.StatusNotFound)

				if rr.Code != http.StatusNotFound {
					require.Equal(t, "application/problem+json", rr.Header().Get("Content-Type"))

					var resp response.BaseResponse
					require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))

					require.Equal(t, false, resp.Ok)
					require.Equal(t, tc.respErr, resp.Error)
				}
			}
		})
	}
}
