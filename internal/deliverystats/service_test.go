package deliverystats

import (
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"go-word-create/internal/jiraservice"
)

type fakeSource struct {
	mu         sync.Mutex
	sprints    []jiraservice.SprintInfo
	velocity   map[int]jiraservice.SprintVelocity
	velErr     error
	bugs       []jiraservice.HistoryIssue
	bugsErr    error
	history    []jiraservice.HistoryIssue
	historyErr error

	byKey map[string]jiraservice.HistoryIssue // answers `key in (...)` lookups

	changelogScans int
	bugSearches    int
	velocityReads  int
	keyQueries     int
}

func (f *fakeSource) ClosedSprints(_ string, limit int) ([]jiraservice.SprintInfo, error) {
	out := f.sprints
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}

func (f *fakeSource) BoardVelocity(string) (map[int]jiraservice.SprintVelocity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.velocityReads++
	return f.velocity, f.velErr
}

func (f *fakeSource) SearchHistoryIssues(jql string, withChangelog bool) ([]jiraservice.HistoryIssue, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if strings.HasPrefix(jql, "key in (") {
		f.keyQueries++
		var out []jiraservice.HistoryIssue
		for _, k := range strings.Split(strings.TrimSuffix(strings.TrimPrefix(jql, "key in ("), ")"), ",") {
			if is, ok := f.byKey[k]; ok {
				out = append(out, is)
			}
		}
		return out, nil
	}
	if strings.Contains(jql, "issuetype = Bug") {
		f.bugSearches++
		return f.bugs, f.bugsErr
	}
	if withChangelog {
		f.changelogScans++
	}
	return f.history, f.historyErr
}

func newFake() *fakeSource {
	f := &fakeSource{velocity: map[int]jiraservice.SprintVelocity{}}
	// Four consecutive two-week sprints starting 2026-09-01.
	start := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	for i := 0; i < 4; i++ {
		s := start.AddDate(0, 0, 14*i)
		id := 100 + i
		f.sprints = append(f.sprints, jiraservice.SprintInfo{ID: id, Name: "S" + string(rune('1'+i)), State: "closed", Start: s, End: s.AddDate(0, 0, 14), Complete: s.AddDate(0, 0, 14)})
		f.velocity[id] = jiraservice.SprintVelocity{SprintID: id, Committed: float64(40 + 10*i), Completed: float64(30 + 10*i)}
	}
	f.bugs = []jiraservice.HistoryIssue{
		{Key: "B-1", Type: "Bug", Priority: "High", Created: start.AddDate(0, 0, 1)},
		{Key: "B-2", Type: "Bug", Priority: "Low", Created: start.AddDate(0, 0, 2), Resolved: start.AddDate(0, 0, 40), Resolution: "Done"},
	}
	f.history = []jiraservice.HistoryIssue{
		{Key: "S-1", Type: "Story", Created: start.AddDate(0, 0, -5), StoryPoints: 8},
		{Key: "S-2", Type: "Story", Created: start.AddDate(0, 0, -5), StoryPoints: 3, Resolved: start.AddDate(0, 0, 3), Resolution: "Done"},
	}
	return f
}

func testService(f *fakeSource, dir string) *Service {
	now := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)
	return New(f, Options{
		ProjectKey: "PRJ", IssueTypes: []string{"Story", "Bug"}, SprintCount: 3, BugDays: 5,
		CacheDir: dir, CacheTTL: time.Hour, Now: func() time.Time { return now },
	})
}

var team = Team{Component: "TEAM", Board: "Team Board"}

func TestReportBuildsEverySection(t *testing.T) {
	f := newFake()
	rep, err := testService(f, t.TempDir()).Report(team, 0, false)
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if rep.Errors != nil {
		t.Fatalf("unexpected errors: %v", rep.Errors)
	}
	if len(rep.Sprints) != 3 || rep.Sprints[0].Name != "S2" || rep.Sprints[2].Name != "S4" {
		t.Fatalf("sprints = %+v, want the last 3 closed (S2..S4)", rep.Sprints)
	}

	// Velocity: S2 completed 40, trailing avg over S1..S2 data that exists (30, 40, 50 for S2 = avg(30,40)).
	v := rep.Velocity[0]
	if *v.Committed != 50 || *v.Completed != 40 || *v.AvgLast3 != 35 {
		t.Errorf("S2 velocity = %v/%v avg %v, want 50/40 avg 35", *v.Committed, *v.Completed, *v.AvgLast3)
	}
	if got := *rep.Velocity[2].AvgLast3; got != 50 {
		t.Errorf("S4 avg last 3 = %v, want 50 (S2..S4 = 40,50,60)", got)
	}
	if got := *rep.CommitmentRate[0].RatePct; got != 80 {
		t.Errorf("S2 commitment = %v, want 80", got)
	}

	if len(rep.BacklogHealth) != 3 || len(rep.LeadCycle) != 3 || len(rep.BugDynamics) != 3 {
		t.Fatalf("per-sprint sections not complete: %d/%d/%d", len(rep.BacklogHealth), len(rep.LeadCycle), len(rep.BugDynamics))
	}
	// At S2's start (day 14) S-1 (8 SP) is still in the backlog; S-2 was resolved on day 3.
	if got := rep.BacklogHealth[0]; got.StoryPoints != 8 || got.Issues != 1 {
		t.Errorf("S2 backlog = %+v, want 8 SP / 1 issue", got)
	}
	if got := rep.BacklogHealth[0].RunwaySprints; got == nil || *got != 0.2 {
		t.Errorf("S2 runway = %v, want 8/35 rounded to 0.2", deref(got))
	}
	if len(rep.OpenBugsDaily) != 5 || rep.OpenBugsDaily[4].Date != "2026-11-03" {
		t.Errorf("daily series = %+v, want 5 days ending 2026-11-03", rep.OpenBugsDaily)
	}
	if rep.OpenBugsDaily[4].Open != 1 {
		t.Errorf("open bugs today = %d, want 1", rep.OpenBugsDaily[4].Open)
	}
	if got := rep.OpenBugsByPriority.Priorities; len(got) != 2 || got[0] != "High" {
		t.Errorf("priorities = %v, want [High Low]", got)
	}
}

func TestClosedSprintHistoryIsServedFromTheDiskCache(t *testing.T) {
	f := newFake()
	dir := t.TempDir()
	if _, err := testService(f, dir).Report(team, 0, false); err != nil {
		t.Fatalf("first Report: %v", err)
	}
	if f.changelogScans != 1 {
		t.Fatalf("first report scanned changelogs %d times, want 1", f.changelogScans)
	}

	// A fresh service (new process) must not rescan closed sprints.
	if _, err := testService(f, dir).Report(team, 0, false); err != nil {
		t.Fatalf("second Report: %v", err)
	}
	if f.changelogScans != 1 {
		t.Errorf("changelog scans = %d after restart, want still 1", f.changelogScans)
	}

	// A newly closed sprint triggers exactly one more scan.
	last := f.sprints[len(f.sprints)-1]
	next := last.Complete
	f.sprints = append(f.sprints, jiraservice.SprintInfo{ID: 104, Name: "S5", State: "closed", Start: next, End: next.AddDate(0, 0, 14), Complete: next.AddDate(0, 0, 14)})
	if _, err := testService(f, dir).Report(team, 0, false); err != nil {
		t.Fatalf("third Report: %v", err)
	}
	if f.changelogScans != 2 {
		t.Errorf("changelog scans = %d after a new sprint closed, want 2", f.changelogScans)
	}
}

func TestMemoryCacheAndRefresh(t *testing.T) {
	f := newFake()
	svc := testService(f, t.TempDir())
	if _, err := svc.Report(team, 0, false); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Report(team, 0, false); err != nil {
		t.Fatal(err)
	}
	if f.bugSearches != 1 {
		t.Errorf("bug searches = %d, want 1 (second call from memory)", f.bugSearches)
	}
	if _, err := svc.Report(team, 0, true); err != nil {
		t.Fatal(err)
	}
	if f.bugSearches != 2 {
		t.Errorf("bug searches = %d after refresh, want 2", f.bugSearches)
	}
	if f.changelogScans != 1 {
		t.Errorf("refresh rescanned closed sprints (%d scans), want 1", f.changelogScans)
	}
}

func TestVelocityHistoryOutlivesJirasSevenSprintWindow(t *testing.T) {
	f := newFake()
	dir := t.TempDir()
	if _, err := testService(f, dir).Report(team, 0, false); err != nil {
		t.Fatal(err)
	}

	// Jira stops reporting the two oldest sprints; the stored values must remain.
	delete(f.velocity, 100)
	delete(f.velocity, 101)
	rep, err := testService(f, dir).Report(team, 4, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Velocity[0].Completed == nil || *rep.Velocity[0].Completed != 30 {
		t.Errorf("S1 completed = %v, want 30 from stored history", deref(rep.Velocity[0].Completed))
	}
}

func TestUnknownVelocityIsNullNotZero(t *testing.T) {
	f := newFake()
	delete(f.velocity, 101)
	rep, err := testService(f, "").Report(team, 3, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Velocity[0].Completed != nil || rep.CommitmentRate[0].RatePct != nil {
		t.Errorf("S2 has no velocity: got %+v / %+v, want nil values", rep.Velocity[0], rep.CommitmentRate[0])
	}
	if rep.Velocity[1].AvgLast3 == nil || *rep.Velocity[1].AvgLast3 != 40 {
		t.Errorf("S3 avg = %v, want 40 (unknown S2 skipped: avg of 30 and 50)", deref(rep.Velocity[1].AvgLast3))
	}
}

func TestSectionFailuresAreIsolated(t *testing.T) {
	f := newFake()
	f.velErr = errors.New("greenhopper unavailable")
	f.bugsErr = errors.New("search failed")
	rep, err := testService(f, t.TempDir()).Report(team, 0, false)
	if err != nil {
		t.Fatalf("Report must not fail as a whole: %v", err)
	}
	for _, key := range []string{"velocity", "commitmentRate", "openBugsDaily", "bugDynamics", "openBugsByPriority"} {
		if !strings.Contains(rep.Errors[key], "TEAM") {
			t.Errorf("errors[%q] = %q, want a message naming the team", key, rep.Errors[key])
		}
	}
	if len(rep.BacklogHealth) != 3 || len(rep.LeadCycle) != 3 {
		t.Errorf("unaffected sections should still be filled, got %d/%d", len(rep.BacklogHealth), len(rep.LeadCycle))
	}
	if rep.BacklogHealth[0].RunwaySprints != nil {
		t.Errorf("runway must be nil without velocity, got %v", *rep.BacklogHealth[0].RunwaySprints)
	}

	f2 := newFake()
	f2.historyErr = errors.New("jira down")
	rep, err = testService(f2, t.TempDir()).Report(team, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Errors["backlogHealth"] == "" || rep.Errors["leadCycle"] == "" || rep.Velocity == nil {
		t.Errorf("history failure should only blank backlog and lead/cycle: %+v", rep.Errors)
	}
}

func TestChangedFingerprintDropsDerivedValuesButKeepsVelocity(t *testing.T) {
	f := newFake()
	dir := t.TempDir()
	if _, err := testService(f, dir).Report(team, 0, false); err != nil {
		t.Fatal(err)
	}
	cache := diskCache{dir: dir}.load(team.Component, "v1|other|settings")
	if len(cache.Sprints) != 0 {
		t.Errorf("derived sprints kept despite a different fingerprint: %d", len(cache.Sprints))
	}
	if len(cache.Velocity) == 0 {
		t.Errorf("velocity history must survive a fingerprint change")
	}
}

func TestSprintCountIsClampedAndCorruptCacheIgnored(t *testing.T) {
	f := newFake()
	dir := t.TempDir()
	if err := os.WriteFile(diskCache{dir: dir}.path(team.Component), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := testService(f, dir).Report(team, 999, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.SprintCount != MaxSprints || len(rep.Sprints) != 4 {
		t.Errorf("count %d with %d sprints; want clamp to %d and all 4 sprints", rep.SprintCount, len(rep.Sprints), MaxSprints)
	}
}

func statusTo(at time.Time, to string) jiraservice.Change {
	return jiraservice.Change{At: at, Field: "status", ToString: to}
}

func TestIssuesClosedWithoutResolutionAreNotOpen(t *testing.T) {
	f := newFake()
	start := f.sprints[0].Start
	closedAt := start.AddDate(0, 0, 20) // inside S2
	// Closed (done category) but never given a resolution; the changelog is only available by key lookup.
	f.bugs = append(f.bugs, jiraservice.HistoryIssue{
		Key: "B-3", Type: "Bug", Priority: "Medium", Status: "Closed", StatusCategory: "done",
		Created: start.AddDate(0, 0, 2),
	})
	f.byKey = map[string]jiraservice.HistoryIssue{
		"B-3": {Key: "B-3", Status: "Closed", StatusCategory: "done", Created: start.AddDate(0, 0, 2),
			Changes: []jiraservice.Change{statusTo(start.AddDate(0, 0, 5), "In Progress"), statusTo(closedAt, "Closed")}},
	}
	// Reopened: resolution still set, but the status is no longer Done.
	f.bugs = append(f.bugs, jiraservice.HistoryIssue{
		Key: "B-4", Type: "Bug", Priority: "High", Status: "Ready for Development", StatusCategory: "new",
		Created: start.AddDate(0, 0, 1), Resolved: start.AddDate(0, 0, 9), Resolution: "Done",
	})

	dir := t.TempDir()
	rep, err := testService(f, dir).Report(team, 3, false)
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if rep.Errors != nil {
		t.Fatalf("unexpected errors: %v", rep.Errors)
	}

	// Today: B-1 (open) and B-4 (reopened) are open; B-2 resolved later, B-3 closed.
	today := rep.OpenBugsDaily[len(rep.OpenBugsDaily)-1]
	if today.Open != 2 {
		t.Errorf("open bugs today = %d, want 2 (B-3 is closed even without a resolution)", today.Open)
	}
	// S2 window is [day 14, day 28): B-3 closed on day 20 counts as fixed there.
	if got := rep.BugDynamics[0]; got.Fixed != 1 {
		t.Errorf("S2 fixed = %d, want 1 (B-3 closed in the sprint)", got.Fixed)
	}
	if f.keyQueries != 1 {
		t.Errorf("key lookups = %d, want 1", f.keyQueries)
	}

	// The close date is remembered: a fresh service and a refresh need no new lookup.
	if _, err := testService(f, dir).Report(team, 3, true); err != nil {
		t.Fatal(err)
	}
	if f.keyQueries != 1 {
		t.Errorf("key lookups = %d after restart, want still 1 (close date cached)", f.keyQueries)
	}
}

func TestSettleResolutionRules(t *testing.T) {
	svc := testService(newFake(), "")
	closed := day(20)
	cache := &cacheData{Closed: map[string]time.Time{"CACHED": day(15)}}
	issues := []jiraservice.HistoryIssue{
		{Key: "KEEP", Status: "Closed", StatusCategory: "done", Resolved: day(5)},
		{Key: "LOG", Status: "Closed", StatusCategory: "done", Changes: []jiraservice.Change{statusTo(day(3), "Closed"), statusTo(day(4), "Open"), statusTo(closed, "Closed")}},
		{Key: "CACHED", Status: "Closed", StatusCategory: "done"},
		{Key: "REOPENED", Status: "Open", StatusCategory: "new", Resolved: day(6)},
		{Key: "UNKNOWN", Status: "?", Resolved: day(7)},
	}
	if err := svc.settleResolution(issues, cache); err != nil {
		t.Fatalf("settleResolution: %v", err)
	}
	want := map[string]time.Time{"KEEP": day(5), "LOG": closed, "CACHED": day(15), "REOPENED": {}, "UNKNOWN": day(7)}
	for _, is := range issues {
		if !is.Resolved.Equal(want[is.Key]) {
			t.Errorf("%s Resolved = %v, want %v", is.Key, is.Resolved, want[is.Key])
		}
	}
}

func TestBacklogIgnoresIssuesClosedWithoutResolution(t *testing.T) {
	f := newFake()
	start := f.sprints[0].Start
	f.history = append(f.history, jiraservice.HistoryIssue{
		Key: "S-3", Type: "Story", Status: "Closed", StatusCategory: "done", StoryPoints: 21,
		Created: start.AddDate(0, 0, -5),
		Changes: []jiraservice.Change{statusTo(start.AddDate(0, 0, 2), "Closed")},
	})
	rep, err := testService(f, t.TempDir()).Report(team, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	// At S2's start (day 14) S-3 has been closed for 12 days, so only S-1's 8 SP remain.
	if got := rep.BacklogHealth[0]; got.StoryPoints != 8 || got.Issues != 1 {
		t.Errorf("S2 backlog = %+v, want 8 SP / 1 issue (S-3 closed without resolution)", got)
	}
}
