package raidscan

import (
	"errors"
	"strings"
	"testing"
	"time"

	"go-word-create/internal/jiraservice"
)

type fakeSource struct {
	active        *jiraservice.SprintInfo
	activeErr     error
	closed        []jiraservice.SprintInfo
	scopes        map[int]*jiraservice.SprintScope
	scopeErr      map[int]error
	scopeCalls    map[int]int
	main          []jiraservice.RiskIssue
	mainErr       error
	sprintKeys    map[string]struct{}
	flaggedKeys   map[string]struct{}
	sprintKeysErr error
	flaggedErr    error
	queries       []string
}

func (f *fakeSource) ActiveSprint(string) (*jiraservice.SprintInfo, error) {
	return f.active, f.activeErr
}
func (f *fakeSource) ClosedSprints(string, int) ([]jiraservice.SprintInfo, error) {
	return f.closed, nil
}

func (f *fakeSource) SprintScope(_ string, id int) (*jiraservice.SprintScope, error) {
	if f.scopeCalls == nil {
		f.scopeCalls = map[int]int{}
	}
	f.scopeCalls[id]++
	if err := f.scopeErr[id]; err != nil {
		return nil, err
	}
	return f.scopes[id], nil
}

func (f *fakeSource) SearchRiskIssues(jql string, withChangelog bool) ([]jiraservice.RiskIssue, error) {
	f.queries = append(f.queries, jql)
	if !withChangelog {
		return nil, errors.New("main scan must read the changelog")
	}
	return f.main, f.mainErr
}

func (f *fakeSource) SearchKeys(jql string) (map[string]struct{}, error) {
	f.queries = append(f.queries, jql)
	if strings.Contains(jql, "flagged") {
		return f.flaggedKeys, f.flaggedErr
	}
	return f.sprintKeys, f.sprintKeysErr
}

func newFake() *fakeSource {
	return &fakeSource{
		active: &jiraservice.SprintInfo{ID: 500, Name: "T26.23", State: "active", Start: testNow.AddDate(0, 0, -7), End: testNow.AddDate(0, 0, 7)},
		closed: []jiraservice.SprintInfo{{ID: 400, Name: "T26.22"}, {ID: 450, Name: "T26.21"}},
		scopes: map[int]*jiraservice.SprintScope{
			400: {Committed: 100, Added: 10, Removed: 5, Reestimated: 5, Final: 110, Completed: 90, CarriedOver: 20, IssuesAdded: 2},
			450: {Committed: 0, Added: 4, Final: 4},
			500: {Committed: 80, Added: 8, Final: 88, Completed: 22},
		},
		main: []jiraservice.RiskIssue{
			issue(func(is *jiraservice.RiskIssue) {
				is.Key, is.Status = "T-1", "Blocked"
				is.Changes = []jiraservice.Change{moved(daysAgo(3), "Blocked")}
			}),
			issue(func(is *jiraservice.RiskIssue) {
				is.Key, is.Status, is.StatusCategory = "T-2", "To Do", "new"
			}),
			issue(func(is *jiraservice.RiskIssue) { // a backlog bug, outside the sprint and not started
				is.Key, is.Type, is.Priority, is.Status, is.StatusCategory = "T-9", "Bug", "Blocker", "To Do", "new"
			}),
		},
		sprintKeys:  map[string]struct{}{"T-1": {}, "T-2": {}},
		flaggedKeys: map[string]struct{}{},
	}
}

func newService(src Source, dir string) *Service {
	return New(src, Options{ProjectKey: "P", CacheDir: dir, CacheTTL: time.Minute, Now: func() time.Time { return testNow }})
}

var team = Team{Component: "ALPHA", Board: "Alpha Team"}

func TestReportBuildsWatchlistScopeAndPulse(t *testing.T) {
	rep := newService(newFake(), "").Report(team, 2, false)

	if rep.Errors != nil || len(rep.Warnings) != 0 {
		t.Fatalf("errors = %v, warnings = %v, want none", rep.Errors, rep.Warnings)
	}
	if rep.Summary.Open != 3 || rep.Summary.Flagged != 2 {
		t.Errorf("open/flagged = %d/%d, want 3/2 (the to-do item is healthy at 50%% of the sprint)", rep.Summary.Open, rep.Summary.Flagged)
	}
	if got := rep.Items[0].Key; got != "T-1" && got != "T-9" {
		t.Errorf("first item = %s, want a high-severity one", got)
	}

	if len(rep.Scope) != 3 || rep.Scope[0].Sprint != "T26.22" || !rep.Scope[2].Active {
		t.Fatalf("scope = %+v, want two closed sprints then the active one", rep.Scope)
	}
	if d := rep.Scope[0].DriftPct; d == nil || *d != 10 {
		t.Errorf("drift of T26.22 = %v, want 10 ((110-100)/100)", d)
	}
	if rep.Scope[1].DriftPct != nil {
		t.Errorf("drift with nothing committed = %v, want nil", *rep.Scope[1].DriftPct)
	}

	p := rep.Pulse
	if p == nil || p.ElapsedPct != 50 || p.DaysLeft != 7 {
		t.Fatalf("pulse = %+v, want 50%% elapsed and 7 days left", p)
	}
	if p.DonePct == nil || *p.DonePct != 25 {
		t.Errorf("DonePct = %v, want 25 (22 of 88)", p.DonePct)
	}
	if p.InProgress != 1 || p.NotStarted != 1 {
		t.Errorf("in progress / not started = %d/%d, want 1/1", p.InProgress, p.NotStarted)
	}
	if rep.Thresholds.AgingDays != 10 || rep.Thresholds.DriftPct != 20 {
		t.Errorf("thresholds = %+v, want the defaults", rep.Thresholds)
	}
}

func TestSprintSummaryCoversOnlyTheRunningSprint(t *testing.T) {
	rep := newService(newFake(), "").Report(team, 2, false)
	all, sp := rep.Summary, rep.SprintSummary
	if sp == nil {
		t.Fatal("SprintSummary is nil with a running sprint")
	}
	if all.Open != 3 || all.Flagged != 2 || all.BySignal[SigCritical] != 1 || all.ByCategory[CatIssue] != 2 {
		t.Errorf("whole-backlog summary = %+v, want 3 open, 2 flagged, the backlog bug counted", all)
	}
	if sp.Open != 2 || sp.Flagged != 1 || sp.BySignal[SigCritical] != 0 || sp.BySignal[SigBlocked] != 1 || sp.ByCategory[CatIssue] != 1 {
		t.Errorf("sprint summary = %+v, want 2 open, 1 flagged (blocked T-1), the backlog bug left out", *sp)
	}
	if sp.ByCategory == nil || sp.HighByCategory == nil {
		t.Error("sprint summary maps must be non-nil so the page can index them")
	}

	noSprint := newFake()
	noSprint.active = nil
	if rep := newService(noSprint, "").Report(team, 2, false); rep.SprintSummary != nil {
		t.Errorf("SprintSummary = %+v, want nil between sprints", *rep.SprintSummary)
	}
}

func TestSprintSummaryIsNilWhenSprintIssuesCannotBeListed(t *testing.T) {
	src := newFake()
	src.sprintKeys = nil
	src.sprintKeysErr = errors.New("jql rejected")
	rep := newService(src, "").Report(team, 2, false)
	if rep.SprintSummary != nil {
		t.Errorf("SprintSummary = %+v, want nil: zeros would claim the sprint is healthy", *rep.SprintSummary)
	}
	if len(rep.Warnings) == 0 {
		t.Error("expected a warning about the unlisted sprint issues")
	}
}

func TestQueryReadsTheWholeOpenBacklogOfTheTeam(t *testing.T) {
	src := newFake()
	newService(src, "").Report(team, 2, false)
	var main string
	for _, q := range src.queries {
		if strings.Contains(q, "issuetype in") {
			main = q
		}
	}
	if strings.Contains(main, "openSprints") {
		t.Errorf("query %q is restricted to sprint work; blocked backlog items and critical bugs would be missed", main)
	}
	for _, want := range []string{`project = P`, `component = "ALPHA"`, `issuetype in ("Bug","Story","Task")`, `statusCategory != Done`} {
		if !strings.Contains(main, want) {
			t.Errorf("main query %q lacks %q", main, want)
		}
	}
}

func TestClosedSprintScopeIsReadOnce(t *testing.T) {
	src := newFake()
	svc := newService(src, t.TempDir())
	svc.Report(team, 2, false)
	svc.Report(team, 2, true) // refresh bypasses the memory copy, not the persisted scopes

	if src.scopeCalls[400] != 1 || src.scopeCalls[450] != 1 {
		t.Errorf("closed sprint reads = %d/%d, want 1/1", src.scopeCalls[400], src.scopeCalls[450])
	}
	if src.scopeCalls[500] != 2 {
		t.Errorf("active sprint reads = %d, want 2 (always live)", src.scopeCalls[500])
	}
	// A new service over the same directory (a restart) still has them.
	src2 := newFake()
	newService(src2, svc.disk.dir).Report(team, 2, false)
	if src2.scopeCalls[400] != 0 {
		t.Errorf("closed sprint read after restart = %d, want 0", src2.scopeCalls[400])
	}
}

func TestMemoryCacheHonorsTTLAndRefresh(t *testing.T) {
	src := newFake()
	now := testNow
	svc := New(src, Options{ProjectKey: "P", CacheTTL: time.Minute, Now: func() time.Time { return now }})
	first := svc.Report(team, 2, false)
	if svc.Report(team, 2, false) != first {
		t.Error("second call within the TTL did not reuse the report")
	}
	if svc.Report(team, 2, true) == first {
		t.Error("refresh reused the in-memory report")
	}
	now = now.Add(2 * time.Minute)
	if svc.Report(team, 2, false) == first {
		t.Error("report older than the TTL was reused")
	}
}

func TestTrendKeepsOneSnapshotPerDay(t *testing.T) {
	src := newFake()
	dir := t.TempDir()
	now := testNow
	svc := New(src, Options{ProjectKey: "P", CacheDir: dir, CacheTTL: time.Minute, Now: func() time.Time { return now }})
	svc.Report(team, 2, true)
	svc.Report(team, 2, true) // same day: replaces, does not append
	now = now.AddDate(0, 0, 1)
	src.main = src.main[:2]
	rep := svc.Report(team, 2, true)

	if len(rep.Trend) != 2 {
		t.Fatalf("trend = %+v, want 2 days", rep.Trend)
	}
	if rep.Trend[0].Date != "2026-11-11" || rep.Trend[1].Date != "2026-11-12" {
		t.Errorf("dates = %s, %s", rep.Trend[0].Date, rep.Trend[1].Date)
	}
	if rep.Trend[0].Blocked != 1 || rep.Trend[0].Open != 3 || rep.Trend[1].Open != 2 {
		t.Errorf("trend = %+v, want open 3 then 2 and one blocked", rep.Trend)
	}
}

func TestFailuresDegradeInsteadOfBlankingThePage(t *testing.T) {
	t.Run("watchlist query fails", func(t *testing.T) {
		src := newFake()
		src.mainErr = errors.New("jira down")
		rep := newService(src, t.TempDir()).Report(team, 2, false)
		if !strings.Contains(rep.Errors["watchlist"], "ALPHA") || !strings.Contains(rep.Errors["watchlist"], "jira down") {
			t.Errorf("watchlist error = %q, want team and cause", rep.Errors["watchlist"])
		}
		if len(rep.Scope) != 3 || rep.Pulse == nil {
			t.Error("scope and pulse must survive a failing watchlist")
		}
		if len(rep.Trend) != 0 {
			t.Errorf("trend = %v, want no snapshot when the watchlist is unknown", rep.Trend)
		}
	})
	t.Run("flagged query fails", func(t *testing.T) {
		src := newFake()
		src.flaggedErr = errors.New("field flagged does not exist")
		rep := newService(src, "").Report(team, 2, false)
		if len(rep.Warnings) != 1 || !strings.Contains(rep.Warnings[0], "flagged") || rep.Errors != nil {
			t.Errorf("warnings = %v, errors = %v, want one warning about flagged", rep.Warnings, rep.Errors)
		}
		if rep.Summary.BySignal[SigBlocked] != 1 {
			t.Error("status-based blocking must still work without the flagged lookup")
		}
	})
	t.Run("one sprint report fails", func(t *testing.T) {
		src := newFake()
		src.scopeErr = map[int]error{400: errors.New("boom")}
		rep := newService(src, "").Report(team, 2, false)
		if len(rep.Scope) != 2 || len(rep.Warnings) != 1 || !strings.Contains(rep.Warnings[0], "T26.22") {
			t.Errorf("scope = %d points, warnings = %v, want the other two sprints and a warning naming T26.22", len(rep.Scope), rep.Warnings)
		}
	})
	t.Run("active sprint cannot be read", func(t *testing.T) {
		src := newFake()
		src.activeErr = errors.New("no board")
		rep := newService(src, "").Report(team, 2, false)
		if rep.Errors["scope"] == "" || rep.Errors["pulse"] == "" {
			t.Errorf("errors = %v, want scope and pulse explained", rep.Errors)
		}
		if rep.Summary.Open == 0 {
			t.Error("the watchlist must still be built without a sprint")
		}
	})
	t.Run("no active sprint", func(t *testing.T) {
		src := newFake()
		src.active = nil
		rep := newService(src, "").Report(team, 2, false)
		if rep.Pulse != nil || rep.Errors != nil {
			t.Errorf("pulse = %v, errors = %v, want neither between sprints", rep.Pulse, rep.Errors)
		}
		if len(rep.Scope) != 2 {
			t.Errorf("scope = %d points, want the 2 closed sprints", len(rep.Scope))
		}
	})
}
