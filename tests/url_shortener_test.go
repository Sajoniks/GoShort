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

func getDefaultClient(t *testing.T) *httpexpect.Expect {
	u := url.URL{Scheme: "http", Host: host}
	return httpexpect.Default(t, u.String())
}

func getNonRedirectClient(t *testing.T) *httpexpect.Expect {
	u := url.URL{Scheme: "http", Host: host}
	return httpexpect.WithConfig(
		httpexpect.Config{
			TestName: t.Name(),
			BaseURL:  u.String(),
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
	e := getDefaultClient(t)

	resp := e.POST("/").
		WithJSON(save.RequestSave{
			URL: "https://www.example.com",
		}).
		Expect().
		Status(http.StatusOK)

	obj := resp.JSON().Object()

	obj.ContainsKey("alias").Value("alias").IsString().NotEqual("")
}

func TestUrlShortener_EmptyData_FailPost(t *testing.T) {
	e := getDefaultClient(t)

	resp := e.POST("/").Expect().Status(http.StatusOK)

	obj := resp.JSON(httpexpect.ContentOpts{MediaType: "application/problem+json"}).Object()

	obj.HasValue("ok", false)
	obj.HasValue("description", "empty request body")
}

func TestUrlShortener_EmptyUrl_FailPost(t *testing.T) {
	e := getDefaultClient(t)

	resp := e.POST("/").
		WithJSON(save.RequestSave{
			URL: " ",
		}).
		Expect().
		Status(http.StatusOK)

	obj := resp.JSON(httpexpect.ContentOpts{
		MediaType: "application/problem+json",
	}).Object()

	obj.HasValue("ok", false)
	obj.HasValue("description", "invalid url")

	resp = e.POST("/").
		WithJSON(save.RequestSave{
			URL: "",
		}).
		Expect().
		Status(http.StatusOK)

	obj = resp.JSON(httpexpect.ContentOpts{
		MediaType: "application/problem+json",
	}).Object()

	obj.HasValue("ok", false)
	obj.HasValue("description", "invalid url")
}

func TestUrlShortener_Redirects(t *testing.T) {
	e := getNonRedirectClient(t)

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
