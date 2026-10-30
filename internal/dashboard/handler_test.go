package dashboard

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-word-create/internal/config"
)

func get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	h := NewHandler(&config.Config{}, nil).Routes()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestFaviconServed(t *testing.T) {
	for _, path := range []string{"/favicon.ico", "/static/favicon.svg"} {
		rec := get(t, path)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "image/svg+xml") {
			t.Errorf("GET %s Content-Type = %q, want image/svg+xml", path, ct)
		}
		if !strings.Contains(rec.Body.String(), "<svg") {
			t.Errorf("GET %s body is not an SVG", path)
		}
	}
}

func TestIndexLinksFavicon(t *testing.T) {
	rec := get(t, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `href="/static/favicon.svg"`) {
		t.Error("index.html does not link /static/favicon.svg")
	}
}
