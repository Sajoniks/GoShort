package sqlite

import (
	urlstore "github.com/sajoniks/GoShort/internal/store/interface"
	"log"
	"os"
	"testing"
)

var store urlstore.CloseableStore

func initDb() {
	_, err := store.SaveURL("www.google.com", "alias")
	if err != nil {
		log.Fatalf("error during testDb init: %v", err)
	}
}

func cleanupDb() {
	store.Close()
	err := os.Remove("testdb.sqlite")
	if err != nil {
		log.Printf("%v", err)
	}
}

func TestMain(m *testing.M) {
	var err error
	store, err = NewSqliteStore("testdb.sqlite")
	if err != nil {
		log.Fatalf("failed to prepare db: %v", err)
	}
	initDb()
	retCode := m.Run()
	cleanupDb()

	os.Exit(retCode)
}

func Test_AddUrl(t *testing.T) {
	_, err := store.SaveURL("www.example.com", "aaa")
	if err != nil {
		t.Errorf("did not want an error: %v", err)
	}
}

func Test_AddDuplicateUrl(t *testing.T) {
	_, err := store.SaveURL("www.site1.com", "aaa")
	if err != nil {
		t.Fatalf("did not want an error: %v", err)
	}
	_, err = store.SaveURL("www.site1.com", "aaa")
	if err == nil {
		t.Errorf("wanted an error, did not get one")
	}
}

func Test_GetUrl(t *testing.T) {
	url, err := store.GetURL("alias")
	if err != nil {
		t.Fatalf("did not want an error: %v", err)
	}
	test := "www.google.com"
	if url != test {
		t.Errorf("want %q, got %q", test, url)
	}
}
