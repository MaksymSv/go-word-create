package deliverystats

import (
	"reflect"
	"testing"
	"time"

	"go-word-create/internal/jiraservice"
)

func day(d int) time.Time { return time.Date(2026, 10, d, 12, 0, 0, 0, time.UTC) }

func change(at time.Time, field, to, toString string) jiraservice.Change {
	return jiraservice.Change{At: at, Field: field, To: to, ToString: toString}
}

func TestTrailingAverageUsesFewerValuesAtTheStart(t *testing.T) {
	vals := []float64{10, 20, 30, 40}
	cases := []struct {
		i    int
		want float64
	}{{0, 10}, {1, 15}, {2, 20}, {3, 30}}
	for _, c := range cases {
		if got := trailingAverage(vals, c.i, 3); got != c.want {
			t.Errorf("trailingAverage(i=%d) = %v, want %v", c.i, got, c.want)
		}
	}
}

func TestSprintWindowDoesNotOverlapThePreviousSprint(t *testing.T) {
	closed := []jiraservice.SprintInfo{
		{ID: 1, Start: day(1), End: day(14), Complete: day(14)},
		{ID: 2, Start: day(14).Add(-2 * time.Hour), End: day(28), Complete: day(28)}, // starts before #1 closed
		{ID: 3, Start: day(30), End: day(31), Complete: day(31)},                     // gap after #2
	}
	if from, to := sprintWindow(closed, 0); !from.Equal(day(1)) || !to.Equal(day(14)) {
		t.Errorf("first window = %v..%v", from, to)
	}
	if from, _ := sprintWindow(closed, 1); !from.Equal(day(14)) {
		t.Errorf("overlapping window starts %v, want previous close %v", from, day(14))
	}
	if from, _ := sprintWindow(closed, 2); !from.Equal(day(30)) {
		t.Errorf("window after a gap starts %v, want its own start %v", from, day(30))
	}
}

func TestEffectiveEndPrefersCompleteDate(t *testing.T) {
	if got := effectiveEnd(jiraservice.SprintInfo{End: day(10), Complete: day(12)}); !got.Equal(day(12)) {
		t.Errorf("got %v, want complete date", got)
	}
	if got := effectiveEnd(jiraservice.SprintInfo{End: day(10)}); !got.Equal(day(10)) {
		t.Errorf("got %v, want planned end", got)
	}
}

func TestBacklogAtReplaysSprintMembership(t *testing.T) {
	types := lowerSet([]string{"Story", "Bug"})
	issues := []jiraservice.HistoryIssue{
		// in backlog for the whole period
		{Key: "A", Type: "Story", Created: day(1), StoryPoints: 5},
		// planned into a sprint on day 5, back to backlog on day 9
		{Key: "B", Type: "Story", Created: day(1), StoryPoints: 3, Changes: []jiraservice.Change{
			change(day(5), "Sprint", "11", "S1"), change(day(9), "Sprint", "", ""),
		}},
		// created after the snapshot
		{Key: "C", Type: "Story", Created: day(8), StoryPoints: 8},
		// resolved before the snapshot
		{Key: "D", Type: "Story", Created: day(1), Resolved: day(4), StoryPoints: 13},
		// excluded type
		{Key: "E", Type: "Epic", Created: day(1), StoryPoints: 21},
		// unestimated
		{Key: "F", Type: "Bug", Created: day(1)},
	}

	sp, n, unest := backlogAt(issues, types, day(6))
	if sp != 5+0 || n != 2 || unest != 1 {
		// B is inside S1 at day 6, so only A (5 SP) and F (no SP) count.
		t.Errorf("backlog at day 6 = %v SP, %d issues, %d unestimated; want 5, 2, 1", sp, n, unest)
	}
	sp, n, unest = backlogAt(issues, types, day(10))
	if sp != 5+3+8 || n != 4 || unest != 1 {
		t.Errorf("backlog at day 10 = %v SP, %d issues, %d unestimated; want 16, 4, 1", sp, n, unest)
	}
}

func TestLeadCycleAveragesOverIssuesDeliveredInTheWindow(t *testing.T) {
	types := lowerSet([]string{"Story"})
	issues := []jiraservice.HistoryIssue{
		// lead 10 d, cycle 4 d
		{Key: "A", Type: "Story", Created: day(1), Resolved: day(11), Resolution: "Done", Changes: []jiraservice.Change{
			change(day(7), "status", "", "In Progress"), change(day(9), "status", "", "In Progress"),
		}},
		// lead 20 d, never In Progress -> counts for lead only
		{Key: "B", Type: "Story", Created: day(1), Resolved: day(21), Resolution: "Done"},
		// outside the window
		{Key: "C", Type: "Story", Created: day(1), Resolved: day(25), Resolution: "Done"},
		// unresolved
		{Key: "D", Type: "Story", Created: day(1)},
		// other type
		{Key: "E", Type: "Bug", Created: day(1), Resolved: day(12), Resolution: "Done"},
	}

	lead, cycle, n, cn := leadCycle(issues, types, nil, day(10), day(22))
	if n != 2 || cn != 1 || lead == nil || *lead != 15 || cycle == nil || *cycle != 4 {
		t.Errorf("leadCycle = lead %v cycle %v n=%d cycleN=%d; want 15, 4, 2, 1", deref(lead), deref(cycle), n, cn)
	}

	lead, cycle, n, _ = leadCycle(issues, types, nil, day(26), day(28))
	if lead != nil || cycle != nil || n != 0 {
		t.Errorf("empty window should give nil averages, got %v %v n=%d", deref(lead), deref(cycle), n)
	}
}

func TestLeadCycleHonoursCompletedResolutions(t *testing.T) {
	types := lowerSet([]string{"Story"})
	issues := []jiraservice.HistoryIssue{
		{Key: "A", Type: "Story", Created: day(1), Resolved: day(11), Resolution: "Done"},
		{Key: "B", Type: "Story", Created: day(1), Resolved: day(21), Resolution: "Duplicate"},
	}
	_, _, n, _ := leadCycle(issues, types, lowerSet([]string{"done", "fixed"}), day(1), day(30))
	if n != 1 {
		t.Errorf("with resolutions {Done, Fixed} n = %d, want 1", n)
	}
	_, _, n, _ = leadCycle(issues, types, nil, day(1), day(30))
	if n != 2 {
		t.Errorf("with no resolution filter n = %d, want 2", n)
	}
}

func TestBugCountsAreHalfOpenAndPointInTime(t *testing.T) {
	bugs := []jiraservice.HistoryIssue{
		{Key: "A", Created: day(2), Resolved: day(8), Resolution: "Done", Priority: "High"},
		{Key: "B", Created: day(8), Priority: "High"},               // created exactly at window end -> next window
		{Key: "C", Created: day(5), Resolved: day(8), Priority: ""}, // resolved exactly at end -> next window
		{Key: "D", Created: day(1)},
	}
	logged, fixed := loggedFixed(bugs, nil, day(2), day(8))
	if logged != 2 || fixed != 0 {
		t.Errorf("window [2,8) logged=%d fixed=%d; want 2, 0", logged, fixed)
	}
	logged, fixed = loggedFixed(bugs, nil, day(8), day(20))
	if logged != 1 || fixed != 2 {
		t.Errorf("window [8,20) logged=%d fixed=%d; want 1, 2", logged, fixed)
	}

	if got := countOpen(bugs, day(6)); got != 3 {
		t.Errorf("open at day 6 = %d, want 3 (A, C, D)", got)
	}
	want := map[string]int{"High": 1, "None": 2}
	if got := openByPriority(bugs, day(6)); !reflect.DeepEqual(got, want) {
		t.Errorf("open by priority at day 6 = %v, want %v", got, want)
	}
}

func TestSortPrioritiesBySeverityThenAlphabetically(t *testing.T) {
	got := sortPriorities([]string{"Low", "Zeta", "Blocker", "Unprioritized", "Medium", "Alpha", "Major"})
	want := []string{"Blocker", "Major", "Medium", "Low", "Unprioritized", "Alpha", "Zeta"}
	// Known names come first in severity order, unknown ones follow alphabetically.
	known := []string{"Blocker", "Major", "Medium", "Low", "Unprioritized"}
	if !reflect.DeepEqual(got[:5], known) || !reflect.DeepEqual(got[5:], []string{"Alpha", "Zeta"}) {
		t.Errorf("sortPriorities = %v, want %v", got, want)
	}
}

func deref(p *float64) interface{} {
	if p == nil {
		return nil
	}
	return *p
}
