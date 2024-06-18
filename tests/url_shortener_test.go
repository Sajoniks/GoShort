package tests

import (
	"github.com/gavv/httpexpect/v2"
	"github.com/sajoniks/GoShort/internal/http-server/handlers/save"
	"net/http"
	"net/url"
	"testing"
)

const (
	host string = "localhost:8080"
)

func getDefaultClient(t *testing.T, url url.URL) *httpexpect.Expect {
	return httpexpect.Default(t, url.String())
}

func getNonRedirectClient(t *testing.T, url url.URL) *httpexpect.Expect {
	return httpexpect.WithConfig(
		httpexpect.Config{
			TestName: t.Name(),
			BaseURL:  url.String(),
			Reporter: httpexpect.NewAssertReporter(t),
			Printers: []httpexpect.Printer{httpexpect.NewCompactPrinter(t)},
			Client: &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			},
		},
	)
}

func TestUrlShortener_SuccessPost(t *testing.T) {
	u := url.URL{Scheme: "http", Host: host}
	e := getDefaultClient(t, u)

	resp := e.POST("/").
		WithJSON(save.RequestSave{
			URL: "https://www.example.com",
		}).
		Expect()

	obj := resp.Status(http.StatusOK).
		HasContentType("application/json").
		JSON().Object()

	obj.ContainsKey("alias").Value("alias").IsString().NotEqual("")
}

func TestUrlShortener_Redirects(t *testing.T) {
	u := url.URL{Scheme: "http", Host: host}
	e := getNonRedirectClient(t, u)

	var alias string

	e.POST("/").
		WithJSON(save.RequestSave{
			URL: "https://www.example.com",
		}).
		Expect().
		JSON().Object().Value("alias").Decode(&alias)

	e.GET("/{0}", alias).
		Expect().
		Status(http.StatusFound).
		Header("Location").IsEqual("https://www.example.com")
}
