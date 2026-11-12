package raidscan

import (
	"testing"
	"time"

	"go-word-create/internal/jiraservice"
)

var testNow = time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)

func daysAgo(n float64) time.Time { return testNow.Add(-time.Duration(n * 24 * float64(time.Hour))) }

func testRules() *rules {
	return &rules{
		opt: Options{AgingDays: 10, StaleDays: 5, RolloverSprints: 3, LateStartPct: 60, DriftPct: 20},
		now: testNow, blockedStatuses: lowerSet([]string{"Blocked", "On Hold"}), blockedLabels: lowerSet([]string{"blocked"}),
		critical: lowerSet([]string{"Blocker", "Critical"}), flagged: map[string]struct{}{}, inSprint: map[string]struct{}{},
		activeSprintID: 500, sprintElapsedPct: 70,
	}
}

func issue(mods ...func(*jiraservice.RiskIssue)) jiraservice.RiskIssue {
	is := jiraservice.RiskIssue{
		HistoryIssue: jiraservice.HistoryIssue{Key: "T-1", Type: "Story", Status: "In Progress", StatusCategory: "indeterminate", StoryPoints: 3, Created: daysAgo(40)},
		Summary:      "do the thing", Assignee: "Ann", Updated: daysAgo(1),
	}
	for _, m := range mods {
		m(&is)
	}
	return is
}

func signalOf(it Item, code string) (Signal, bool) {
	for _, s := range it.Signals {
		if s.Code == code {
			return s, true
		}
	}
	return Signal{}, false
}

func codes(it Item) []string {
	var out []string
	for _, s := range it.Signals {
		out = append(out, s.Code)
	}
	return out
}

func moved(at time.Time, to string) jiraservice.Change {
	return jiraservice.Change{At: at, Field: "status", ToString: to}
}

func TestHealthyIssueHasNoSignals(t *testing.T) {
	it := testRules().evaluate(issue(func(is *jiraservice.RiskIssue) {
		is.Changes = []jiraservice.Change{moved(daysAgo(3), "In Progress")}
	}))
	if len(it.Signals) != 0 {
		t.Errorf("signals = %v, want none", codes(it))
	}
	if it.AgeDays == nil || *it.AgeDays != 3 {
		t.Errorf("AgeDays = %v, want 3", it.AgeDays)
	}
}

func TestBlockedByStatusLabelAndFlag(t *testing.T) {
	r := testRules()
	byStatus := r.evaluate(issue(func(is *jiraservice.RiskIssue) {
		is.Status = "Blocked"
		is.Changes = []jiraservice.Change{moved(daysAgo(9), "In Progress"), moved(daysAgo(6), "Blocked")}
	}))
	if s, ok := signalOf(byStatus, SigBlocked); !ok || s.Label != "Blocked 6d" || s.Severity != SevHigh || s.Category != CatIssue {
		t.Errorf("status block = %+v (found %v), want label %q, high, category I", s, ok, "Blocked 6d")
	}

	byLabel := r.evaluate(issue(func(is *jiraservice.RiskIssue) {
		is.Labels = []string{"Blocked"}
		is.Changes = []jiraservice.Change{{At: daysAgo(2), Field: "labels", FromString: "x", ToString: "x blocked"}}
	}))
	if s, ok := signalOf(byLabel, SigBlocked); !ok || s.Label != "Blocked 2d" {
		t.Errorf("label block = %+v (found %v), want label %q", s, ok, "Blocked 2d")
	}

	r.flagged["T-1"] = struct{}{}
	flagged := r.evaluate(issue(func(is *jiraservice.RiskIssue) {
		is.Changes = []jiraservice.Change{{At: daysAgo(4), Field: "Flagged", ToString: "Impediment"}}
	}))
	if s, ok := signalOf(flagged, SigBlocked); !ok || s.Label != "Blocked 4d" {
		t.Errorf("flag block = %+v (found %v), want label %q", s, ok, "Blocked 4d")
	}

	unknownSince := r.evaluate(issue(func(is *jiraservice.RiskIssue) { is.Status = "On Hold" }))
	if s, ok := signalOf(unknownSince, SigBlocked); !ok || s.Label != "Blocked" {
		t.Errorf("block without history = %+v (found %v), want a plain %q label", s, ok, "Blocked")
	}
}

func TestCriticalBugOnlyForBugsWithCriticalPriority(t *testing.T) {
	r := testRules()
	bug := r.evaluate(issue(func(is *jiraservice.RiskIssue) { is.Type, is.Priority = "Bug", "critical" }))
	if _, ok := signalOf(bug, SigCritical); !ok {
		t.Error("critical bug not flagged (priority match must ignore case)")
	}
	story := r.evaluate(issue(func(is *jiraservice.RiskIssue) { is.Priority = "Critical" }))
	if _, ok := signalOf(story, SigCritical); ok {
		t.Error("critical story flagged as critical bug")
	}
	minor := r.evaluate(issue(func(is *jiraservice.RiskIssue) { is.Type, is.Priority = "Bug", "Minor" }))
	if _, ok := signalOf(minor, SigCritical); ok {
		t.Error("minor bug flagged as critical bug")
	}
}

func TestDependencyIgnoresFinishedBlockers(t *testing.T) {
	r := testRules()
	it := r.evaluate(issue(func(is *jiraservice.RiskIssue) {
		is.Links = []jiraservice.IssueLink{
			{Relation: "is blocked by", Key: "X-1", StatusCategory: "indeterminate"},
			{Relation: "is blocked by", Key: "X-2", StatusCategory: "done"},
			{Relation: "blocks", Key: "X-3", StatusCategory: "new"},
			{Relation: "relates to", Key: "X-4"},
		}
	}))
	s, ok := signalOf(it, SigDependency)
	if !ok || s.Label != "Waits for X-1" || s.Category != CatDependency {
		t.Errorf("dependency = %+v (found %v), want only X-1", s, ok)
	}
	two := r.evaluate(issue(func(is *jiraservice.RiskIssue) {
		is.Links = []jiraservice.IssueLink{{Relation: "is blocked by", Key: "X-1"}, {Relation: "depends on", Key: "X-2"}}
	}))
	if s, _ := signalOf(two, SigDependency); s.Label != "Waits for 2 issues" {
		t.Errorf("label = %q, want %q", s.Label, "Waits for 2 issues")
	}
}

func TestAgingAndStaleThresholds(t *testing.T) {
	r := testRules()
	age := func(n float64, updatedAgo float64) Item {
		return r.evaluate(issue(func(is *jiraservice.RiskIssue) {
			is.Changes = []jiraservice.Change{moved(daysAgo(n), "In Progress")}
			is.Updated = daysAgo(updatedAgo)
		}))
	}
	if _, ok := signalOf(age(9.9, 0), SigAging); ok {
		t.Error("aging flagged below the limit")
	}
	if s, ok := signalOf(age(10, 0), SigAging); !ok || s.Severity != SevMedium {
		t.Errorf("aging at the limit = %+v (found %v), want medium", s, ok)
	}
	if s, ok := signalOf(age(20, 0), SigAging); !ok || s.Severity != SevHigh {
		t.Errorf("aging at twice the limit = %+v (found %v), want high", s, ok)
	}
	if _, ok := signalOf(age(2, 4.9), SigStale); ok {
		t.Error("stale flagged below the limit")
	}
	if s, ok := signalOf(age(2, 6), SigStale); !ok || s.Label != "No update 6d" {
		t.Errorf("stale = %+v (found %v), want label %q", s, ok, "No update 6d")
	}
	notStarted := r.evaluate(issue(func(is *jiraservice.RiskIssue) {
		is.Status, is.StatusCategory, is.Created, is.Updated = "To Do", "new", daysAgo(50), daysAgo(50)
	}))
	if _, ok := signalOf(notStarted, SigAging); ok {
		t.Error("an issue that has not started must not age")
	}
	if _, ok := signalOf(notStarted, SigStale); ok {
		t.Error("an issue that has not started must not be stale")
	}
}

func TestAgeStartsAtFirstInProgress(t *testing.T) {
	it := testRules().evaluate(issue(func(is *jiraservice.RiskIssue) {
		is.Changes = []jiraservice.Change{moved(daysAgo(30), "Ready for Dev"), moved(daysAgo(12), "In Progress"), moved(daysAgo(11), "In Review"), moved(daysAgo(1), "In Progress")}
		is.Updated = daysAgo(0)
	}))
	if it.AgeDays == nil || *it.AgeDays != 12 {
		t.Errorf("AgeDays = %v, want 12 (first move to In Progress)", it.AgeDays)
	}
	if it.StatusDays == nil || *it.StatusDays != 1 {
		t.Errorf("StatusDays = %v, want 1 (last status change)", it.StatusDays)
	}
}

func TestLateStartNeedsSprintItemAndElapsedTime(t *testing.T) {
	r := testRules()
	todo := func(is *jiraservice.RiskIssue) { is.Status, is.StatusCategory = "To Do", "new" }
	r.inSprint["T-1"] = struct{}{}
	if s, ok := signalOf(r.evaluate(issue(todo)), SigLateStart); !ok || s.Severity != SevMedium {
		t.Errorf("late start = %+v (found %v), want medium", s, ok)
	}
	r.sprintElapsedPct = 59
	if _, ok := signalOf(r.evaluate(issue(todo)), SigLateStart); ok {
		t.Error("late start flagged before the elapsed share")
	}
	r.sprintElapsedPct = 90
	delete(r.inSprint, "T-1")
	if _, ok := signalOf(r.evaluate(issue(todo)), SigLateStart); ok {
		t.Error("late start flagged for an issue outside the sprint")
	}
	r.inSprint["T-1"] = struct{}{}
	r.sprintElapsedPct = -1
	if _, ok := signalOf(r.evaluate(issue(todo)), SigLateStart); ok {
		t.Error("late start flagged without a running sprint")
	}
}

func TestRolledOverCountsDistinctSprints(t *testing.T) {
	r := testRules()
	sprintChange := func(from, to string) jiraservice.Change {
		return jiraservice.Change{At: daysAgo(20), Field: "Sprint", From: from, To: to}
	}
	two := r.evaluate(issue(func(is *jiraservice.RiskIssue) {
		is.Changes = []jiraservice.Change{sprintChange("", "100"), sprintChange("100", "100, 200")}
	}))
	if _, ok := signalOf(two, SigRolledOver); ok {
		t.Error("two sprints flagged as rolled over")
	}
	three := r.evaluate(issue(func(is *jiraservice.RiskIssue) {
		is.Changes = []jiraservice.Change{sprintChange("", "100"), sprintChange("100", "100, 200"), sprintChange("100, 200", "200, 300")}
	}))
	if s, ok := signalOf(three, SigRolledOver); !ok || s.Label != "In 3 sprints" {
		t.Errorf("rolled over = %+v (found %v), want label %q", s, ok, "In 3 sprints")
	}
	r.inSprint["T-1"] = struct{}{}
	inSprint := r.evaluate(issue(func(is *jiraservice.RiskIssue) {
		is.Status, is.StatusCategory = "To Do", "new"
		is.Changes = []jiraservice.Change{sprintChange("", "100"), sprintChange("100", "200"), sprintChange("200", "300")}
	}))
	if _, ok := signalOf(inSprint, SigRolledOver); !ok {
		t.Error("a sprint item that went through three sprints must count as rolled over")
	}
	delete(r.inSprint, "T-1")
	idle := r.evaluate(issue(func(is *jiraservice.RiskIssue) {
		is.Status, is.StatusCategory = "To Do", "new"
		is.Changes = []jiraservice.Change{sprintChange("", "100"), sprintChange("100", "200"), sprintChange("200", "300")}
	}))
	if _, ok := signalOf(idle, SigRolledOver); ok {
		t.Error("an idle backlog item that once went through sprints is not rolling over")
	}
	r.inSprint["T-1"] = struct{}{}
	withActive := r.evaluate(issue(func(is *jiraservice.RiskIssue) {
		is.Changes = []jiraservice.Change{sprintChange("", "100"), sprintChange("100", "100, 200")}
	}))
	if _, ok := signalOf(withActive, SigRolledOver); !ok {
		t.Error("the active sprint must count towards the rollover total")
	}
}

func TestOverdue(t *testing.T) {
	r := testRules()
	late := r.evaluate(issue(func(is *jiraservice.RiskIssue) { is.DueDate = daysAgo(3) }))
	if s, ok := signalOf(late, SigOverdue); !ok || s.Severity != SevHigh {
		t.Errorf("overdue = %+v (found %v), want high", s, ok)
	}
	today := r.evaluate(issue(func(is *jiraservice.RiskIssue) { is.DueDate = time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC) }))
	if _, ok := signalOf(today, SigOverdue); ok {
		t.Error("an issue due today is not overdue yet")
	}
}

func TestAssumptionSignals(t *testing.T) {
	r := testRules()
	r.inSprint["T-1"] = struct{}{}
	story := r.evaluate(issue(func(is *jiraservice.RiskIssue) { is.StoryPoints, is.Assignee = 0, "" }))
	if s, ok := signalOf(story, SigUnestimated); !ok || s.Severity != SevLow || s.Category != CatAssumption {
		t.Errorf("unestimated = %+v (found %v), want low, category A", s, ok)
	}
	if _, ok := signalOf(story, SigUnassigned); !ok {
		t.Error("an unassigned item in progress must be flagged")
	}
	bug := r.evaluate(issue(func(is *jiraservice.RiskIssue) { is.Type, is.StoryPoints = "Bug", 0 }))
	if _, ok := signalOf(bug, SigUnestimated); ok {
		t.Error("bugs are not expected to carry an estimate")
	}
	todo := r.evaluate(issue(func(is *jiraservice.RiskIssue) { is.Status, is.StatusCategory, is.Assignee = "To Do", "new", "" }))
	if _, ok := signalOf(todo, SigUnassigned); ok {
		t.Error("an unassigned to-do item is normal (it has not been picked up)")
	}
	r.inSprint = map[string]struct{}{}
	outside := r.evaluate(issue(func(is *jiraservice.RiskIssue) { is.StoryPoints = 0 }))
	if _, ok := signalOf(outside, SigUnestimated); ok {
		t.Error("unestimated flagged outside the sprint")
	}
}

func TestRankSummaryAndBuckets(t *testing.T) {
	r := testRules()
	blocked := r.evaluate(issue(func(is *jiraservice.RiskIssue) {
		is.Key, is.Status = "T-1", "Blocked"
		is.Changes = []jiraservice.Change{moved(daysAgo(2), "Blocked")}
	}))
	aging := r.evaluate(issue(func(is *jiraservice.RiskIssue) {
		is.Key = "T-2"
		is.Changes = []jiraservice.Change{moved(daysAgo(12), "In Progress")}
	}))
	stale := r.evaluate(issue(func(is *jiraservice.RiskIssue) {
		is.Key, is.Updated = "T-3", daysAgo(8)
		is.Changes = []jiraservice.Change{moved(daysAgo(3), "In Progress")}
	}))
	items := []Item{stale, aging, blocked}
	rank(items)
	if got := []string{items[0].Key, items[1].Key, items[2].Key}; got[0] != "T-1" || got[1] != "T-2" || got[2] != "T-3" {
		t.Errorf("rank = %v, want [T-1 T-2 T-3] (blocked, then aging, then stale)", got)
	}

	sum := summarize(10, items)
	if sum.Open != 10 || sum.Flagged != 3 {
		t.Errorf("open/flagged = %d/%d, want 10/3", sum.Open, sum.Flagged)
	}
	if sum.ByCategory[CatIssue] != 1 || sum.ByCategory[CatRisk] != 2 || sum.ByCategory[CatDependency] != 0 {
		t.Errorf("ByCategory = %v, want I:1 R:2 D:0", sum.ByCategory)
	}
	if sum.HighByCategory[CatIssue] != 1 || sum.HighByCategory[CatRisk] != 0 {
		t.Errorf("HighByCategory = %v, want I:1 R:0", sum.HighByCategory)
	}
	if sum.BySignal[SigBlocked] != 1 || sum.BySignal[SigAging] != 1 || sum.BySignal[SigStale] != 1 {
		t.Errorf("BySignal = %v", sum.BySignal)
	}

	buckets := agingBuckets([]jiraservice.RiskIssue{
		issue(func(is *jiraservice.RiskIssue) { is.Changes = []jiraservice.Change{moved(daysAgo(1), "In Progress")} }),
		issue(func(is *jiraservice.RiskIssue) { is.Changes = []jiraservice.Change{moved(daysAgo(5), "In Progress")} }),
		issue(func(is *jiraservice.RiskIssue) { is.Changes = []jiraservice.Change{moved(daysAgo(12), "In Progress")} }),
		issue(func(is *jiraservice.RiskIssue) { is.Changes = []jiraservice.Change{moved(daysAgo(100), "In Progress")} }),
		issue(func(is *jiraservice.RiskIssue) { is.Status, is.StatusCategory = "To Do", "new" }),
	}, testNow)
	want := []int{1, 1, 1, 0, 1}
	for i, b := range buckets {
		if b.Count != want[i] {
			t.Errorf("bucket %q = %d, want %d", b.Label, b.Count, want[i])
		}
	}
}
