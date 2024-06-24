package stress

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/brianvoe/gofakeit"
	"github.com/sajoniks/GoShort/internal/http-server/handlers/save"
	"net/http"
	"runtime"
	"sync"
	"testing"
	"time"
)

const host = "http://localhost:8080"

func Test_GetUrl_30s(t *testing.T) {
	t.Logf("Run %d threads", runtime.GOMAXPROCS(0))
	var wg sync.WaitGroup

	ctx, _ := context.WithTimeout(context.Background(), time.Second*30)

	wg.Add(runtime.GOMAXPROCS(0))
	for i := 0; i < runtime.GOMAXPROCS(0)-1; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					wg.Add(1)
					go func() {
						defer wg.Done()
						u := host + "/" + gofakeit.BuzzWord()
						_, err := http.Get(u) // some garbage
						if err != nil {
							t.Logf("goroutine error: %v", err)
						} else {
							t.Logf("get %s", u)
						}
					}()
					time.Sleep(time.Millisecond * 100)
				}
			}
		}()
	}

	wg.Wait()
	t.Log("testing done")
}

func Test_PostUrl_30s(t *testing.T) {
	t.Logf("Run %d threads", runtime.GOMAXPROCS(0))
	var wg sync.WaitGroup

	ctx, _ := context.WithTimeout(context.Background(), time.Second*30)
	c := make(chan string, 1)

	wg.Add(runtime.GOMAXPROCS(0))
	for i := 0; i < runtime.GOMAXPROCS(0)-1; i++ {
		go func() {
			defer wg.Done()
			for v := range c {
				buf := &bytes.Buffer{}
				req := save.RequestSave{
					URL: v,
				}
				_ = json.NewEncoder(buf).Encode(&req)

				wg.Add(1)
				go func() {
					defer wg.Done()
					_, err := http.Post(host+"/", "application/json", buf)
					if err != nil {
						t.Logf("goroutine error: %v", err)
					} else {
						t.Logf("sent %v", req)
					}
				}()
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				close(c)
				return
			default:
				c <- gofakeit.URL()
				time.Sleep(time.Millisecond * 100)
			}
		}
	}()
	wg.Wait()

	t.Log("stopped testing")
}
