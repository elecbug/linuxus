package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elecbug/linuxus/src/static"
)

func TestStaticAssetsWithoutSourceDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	a := &App{mux: http.NewServeMux()}
	a.handleFavicon()
	want, err := static.Files.ReadFile("favicon.png")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/favicon.ico", "/static/favicon.png"} {
		rr := httptest.NewRecorder()
		a.mux.ServeHTTP(rr, httptest.NewRequest("GET", path, nil))
		if rr.Code != http.StatusOK || !bytes.Equal(rr.Body.Bytes(), want) {
			t.Fatalf("asset %s not served: %d", path, rr.Code)
		}
	}
}
