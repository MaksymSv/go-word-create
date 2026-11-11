package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-word-create/internal/config"
	"go-word-create/internal/deliverystats"
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

func getWithTeams(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	cfg := &config.Config{
		Teams:               []config.TeamEntry{{ComponentName: "ALPHA", BoardName: "Alpha Team"}},
		DeliverySprintCount: 4,
		DeliveryBugDays:     30,
	}
	rec := httptest.NewRecorder()
	NewHandler(cfg, nil).Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestDeliveryPageAndAssetsServed(t *testing.T) {
	rec := get(t, "/delivery")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /delivery = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{`href="/static/favicon.svg"`, `src="/static/charts.js"`, `src="/static/delivery.js"`, `href="/static/delivery.css"`} {
		if !strings.Contains(body, want) {
			t.Errorf("delivery.html is missing %s", want)
		}
	}
	for _, path := range []string{"/static/charts.js", "/static/delivery.js", "/static/delivery.css"} {
		if rec := get(t, path); rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, rec.Code)
		}
	}
}

func TestBothPagesLinkToEachOther(t *testing.T) {
	if body := get(t, "/").Body.String(); !strings.Contains(body, `href="/delivery"`) {
		t.Error("index.html has no link to /delivery")
	}
	if body := get(t, "/delivery").Body.String(); !strings.Contains(body, `href="/"`) {
		t.Error("delivery.html has no link back to /")
	}
}

func TestDeliverySettings(t *testing.T) {
	rec := getWithTeams(t, "/api/delivery/settings")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/delivery/settings = %d, want 200", rec.Code)
	}
	var got DeliverySettings
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if got.SprintCount != 4 || got.BugDays != 30 || got.MaxSprints != deliverystats.MaxSprints {
		t.Errorf("settings = %+v, want sprintCount 4, bugDays 30, maxSprints %d", got, deliverystats.MaxSprints)
	}
}

func TestDeliveryReportRejectsBadRequests(t *testing.T) {
	cases := []struct {
		path string
		want int
	}{
		{"/api/delivery/UNKNOWN", http.StatusNotFound},
		{"/api/delivery/ALPHA?sprints=abc", http.StatusBadRequest},
		{"/api/delivery/ALPHA?sprints=0", http.StatusBadRequest},
		{"/api/delivery/ALPHA?sprints=999", http.StatusBadRequest},
	}
	for _, c := range cases {
		if rec := getWithTeams(t, c.path); rec.Code != c.want {
			t.Errorf("GET %s = %d, want %d (body %s)", c.path, rec.Code, c.want, rec.Body.String())
		}
	}
}
