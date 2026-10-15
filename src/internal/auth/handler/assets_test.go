package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/elecbug/linuxus/src/static"
)

func TestStaticAssetsWithoutSourceDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	a := &App{mux: http.NewServeMux()}
	a.handleFavicon()
	for _, tc := range []struct{ path, name, contentType string }{
		{"/favicon.ico", "favicon.png", "image/png"},
		{"/static/favicon.png", "favicon.png", "image/png"},
		{"/static/ui.css", "ui.css", "text/css"},
		{"/static/workspace.js", "workspace.js", "javascript"},
	} {
		want, err := static.Files.ReadFile(tc.name)
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		a.mux.ServeHTTP(rr, httptest.NewRequest("GET", tc.path, nil))
		if rr.Code != http.StatusOK || !bytes.Equal(rr.Body.Bytes(), want) {
			t.Fatalf("asset %s not served: %d", tc.path, rr.Code)
		}
		if !strings.Contains(rr.Header().Get("Content-Type"), tc.contentType) {
			t.Fatalf("incorrect asset content type: %s", tc.path)
		}
		if strings.HasPrefix(tc.path, "/static/") && rr.Header().Get("Cache-Control") != "no-cache" {
			t.Fatalf("UI asset can remain stale: %s", tc.path)
		}
	}
}
