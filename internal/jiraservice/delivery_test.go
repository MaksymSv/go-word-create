package jiraservice

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jira "github.com/andygrunwald/go-jira"
)

func newTestService(t *testing.T, mux *http.ServeMux) *JiraService {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	client, err := jira.NewClient(srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("jira.NewClient: %v", err)
	}
	return &JiraService{client: client, spField: "customfield_10004", url: srv.URL}
}

func jsonHandler(v interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}
}

func boardMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/agile/1.0/board", jsonHandler(map[string]interface{}{
		"isLast": true, "values": []map[string]interface{}{{"id": 7, "name": "Team Board", "type": "scrum"}},
	}))
	return mux
}

func TestClosedSprintsAreChronologicalNotByID(t *testing.T) {
	mux := boardMux()
	// ID order differs from start-date order, as seen on real boards.
	mux.HandleFunc("/rest/agile/1.0/board/7/sprint", jsonHandler(map[string]interface{}{
		"isLast": true,
		"values": []map[string]interface{}{
			{"id": 30, "name": "S2", "state": "closed", "startDate": "2026-08-05T06:00:00.000Z", "endDate": "2026-08-12T06:00:00.000Z", "completeDate": "2026-08-12T07:00:00.000Z"},
			{"id": 40, "name": "S1", "state": "closed", "startDate": "2026-07-15T06:00:00.000Z", "endDate": "2026-07-29T06:00:00.000Z", "completeDate": "2026-07-31T07:00:00.000Z"},
			{"id": 50, "name": "S3", "state": "closed", "startDate": "2026-08-12T06:00:00.000Z", "endDate": "2026-08-26T06:00:00.000Z"},
			{"id": 60, "name": "Active", "state": "active", "startDate": "2026-08-26T06:00:00.000Z"},
			{"id": 20, "name": "NoStart", "state": "closed"},
		},
	}))
	svc := newTestService(t, mux)

	all, err := svc.ClosedSprints("Team Board", 0)
	if err != nil {
		t.Fatalf("ClosedSprints: %v", err)
	}
	var names []string
	for _, s := range all {
		names = append(names, s.Name)
	}
	if got, want := names, []string{"S1", "S2", "S3"}; len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("closed sprints = %v, want %v (active and undated sprints excluded)", got, want)
	}
	if !all[0].Complete.Equal(time.Date(2026, 7, 31, 7, 0, 0, 0, time.UTC)) {
		t.Errorf("S1 Complete = %v, want 2026-07-31T07:00Z", all[0].Complete)
	}

	newest, err := svc.ClosedSprints("Team Board", 2)
	if err != nil {
		t.Fatalf("ClosedSprints(limit): %v", err)
	}
	if len(newest) != 2 || newest[0].Name != "S2" || newest[1].Name != "S3" {
		t.Errorf("limit 2 = %+v, want the newest two (S2, S3) oldest first", newest)
	}
}

func TestBoardVelocityParsesCommittedAndCompleted(t *testing.T) {
	mux := boardMux()
	mux.HandleFunc("/rest/greenhopper/1.0/rapid/charts/velocity", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("rapidViewId"); got != "7" {
			t.Errorf("rapidViewId = %q, want 7", got)
		}
		jsonHandler(map[string]interface{}{
			"sprints": []map[string]interface{}{{"id": 11, "name": "S1"}, {"id": 12, "name": "S2"}},
			"velocityStatEntries": map[string]interface{}{
				"11": map[string]interface{}{"estimated": map[string]float64{"value": 40}, "completed": map[string]float64{"value": 35.5}},
			},
		})(w, r)
	})
	svc := newTestService(t, mux)

	got, err := svc.BoardVelocity("Team Board")
	if err != nil {
		t.Fatalf("BoardVelocity: %v", err)
	}
	if v := got[11]; v.Committed != 40 || v.Completed != 35.5 || v.Name != "S1" {
		t.Errorf("sprint 11 = %+v, want committed 40 completed 35.5", v)
	}
	if v, ok := got[12]; !ok || v.Committed != 0 || v.Completed != 0 {
		t.Errorf("sprint 12 = %+v ok=%v, want present with zero values", v, ok)
	}
}

func TestToHistoryIssueMapsFieldsAndSortsChanges(t *testing.T) {
	svc := &JiraService{spField: "customfield_10004"}
	created := jira.Time(time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC))
	is := jira.Issue{
		Key: "ABC-1",
		Fields: &jira.IssueFields{
			Type:       jira.IssueType{Name: "Bug"},
			Status:     &jira.Status{Name: "Closed", StatusCategory: jira.StatusCategory{Key: "done"}},
			Priority:   &jira.Priority{Name: "High"},
			Resolution: &jira.Resolution{Name: "Done"},
			Created:    created,
			Unknowns:   map[string]interface{}{"customfield_10004": 5.0},
		},
		Changelog: &jira.Changelog{Histories: []jira.ChangelogHistory{
			{Created: "2026-05-03T10:00:00.000+0000", Items: []jira.ChangelogItems{{Field: "status", ToString: "Done"}}},
			{Created: "2026-05-02T10:00:00.000+0000", Items: []jira.ChangelogItems{{Field: "Sprint", From: nil, To: "981406", ToString: "S1"}}},
			{Created: "garbage", Items: []jira.ChangelogItems{{Field: "status", ToString: "ignored"}}},
		}},
	}

	h := svc.toHistoryIssue(is)
	if h.Key != "ABC-1" || h.Type != "Bug" || h.Priority != "High" || h.Resolution != "Done" || h.StoryPoints != 5 {
		t.Errorf("unexpected mapping: %+v", h)
	}
	if h.Status != "Closed" || h.StatusCategory != "done" {
		t.Errorf("status = %q category %q, want Closed / done", h.Status, h.StatusCategory)
	}
	if !h.Resolved.IsZero() {
		t.Errorf("Resolved = %v, want zero when resolutiondate is absent", h.Resolved)
	}
	if len(h.Changes) != 2 {
		t.Fatalf("changes = %d, want 2 (unparseable timestamp skipped)", len(h.Changes))
	}
	if h.Changes[0].Field != "Sprint" || h.Changes[0].To != "981406" || h.Changes[0].From != "" {
		t.Errorf("first change = %+v, want the earlier Sprint change with empty From", h.Changes[0])
	}
	if h.Changes[1].Field != "status" {
		t.Errorf("second change = %+v, want the later status change", h.Changes[1])
	}
}
