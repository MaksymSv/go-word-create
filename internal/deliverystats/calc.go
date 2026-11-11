package deliverystats

import (
	"math"
	"sort"
	"strings"
	"time"

	"go-word-create/internal/jiraservice"
)

// effectiveEnd is the close time of a finished sprint, else its planned end.
func effectiveEnd(s jiraservice.SprintInfo) time.Time {
	if !s.Complete.IsZero() {
		return s.Complete
	}
	return s.End
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

func ptr(v float64) *float64 { return &v }

func lowerSet(items []string) map[string]struct{} {
	m := make(map[string]struct{}, len(items))
	for _, it := range items {
		m[strings.ToLower(strings.TrimSpace(it))] = struct{}{}
	}
	return m
}

func inSet(set map[string]struct{}, v string) bool {
	_, ok := set[strings.ToLower(strings.TrimSpace(v))]
	return ok
}

// delivered reports whether the issue was resolved with a resolution that
// counts as delivered; an empty set accepts every resolution.
func delivered(is jiraservice.HistoryIssue, completed map[string]struct{}) bool {
	if is.Resolved.IsZero() {
		return false
	}
	return len(completed) == 0 || inSet(completed, is.Resolution)
}

// sprintWindow is the interval [from, to) whose events belong to closed[i].
// Consecutive Jira sprints can overlap by a few hours, so the window starts at
// the later of the sprint start and the previous sprint's end.
func sprintWindow(closed []jiraservice.SprintInfo, i int) (from, to time.Time) {
	from = closed[i].Start
	if i > 0 {
		if prev := effectiveEnd(closed[i-1]); prev.After(from) {
			from = prev
		}
	}
	return from, effectiveEnd(closed[i])
}

// trailingAverage returns the mean of vals[i-window+1..i], using fewer values near the start.
func trailingAverage(vals []float64, i, window int) float64 {
	from := i - window + 1
	if from < 0 {
		from = 0
	}
	sum := 0.0
	for _, v := range vals[from : i+1] {
		sum += v
	}
	return sum / float64(i-from+1)
}

// openAt reports whether the issue existed and was unresolved at the given time.
func openAt(is jiraservice.HistoryIssue, at time.Time) bool {
	if is.Created.After(at) {
		return false
	}
	return is.Resolved.IsZero() || is.Resolved.After(at)
}

// inSprintAt replays the Sprint field changes: after the last change not later
// than `at`, the issue belongs to a sprint when the new value is non-empty.
func inSprintAt(is jiraservice.HistoryIssue, at time.Time) bool {
	member := false
	for _, c := range is.Changes {
		if c.At.After(at) {
			break
		}
		if c.Field == "Sprint" {
			member = strings.TrimSpace(c.To) != ""
		}
	}
	return member
}

// backlogAt sums the story points of issues that were open and in no sprint at `at`.
func backlogAt(issues []jiraservice.HistoryIssue, types map[string]struct{}, at time.Time) (sp float64, n, unestimated int) {
	for _, is := range issues {
		if !inSet(types, is.Type) || !openAt(is, at) || inSprintAt(is, at) {
			continue
		}
		n++
		if is.StoryPoints <= 0 {
			unestimated++
		}
		sp += is.StoryPoints
	}
	return sp, n, unestimated
}

// inProgressAt returns the time of the first transition into "In Progress".
func inProgressAt(is jiraservice.HistoryIssue) (time.Time, bool) {
	for _, c := range is.Changes {
		if c.Field == "status" && strings.EqualFold(strings.TrimSpace(c.ToString), "In Progress") {
			return c.At, true
		}
	}
	return time.Time{}, false
}

// leadCycle averages lead time (created -> resolved) and cycle time
// (first In Progress -> resolved), in days, over issues delivered in [start, end).
func leadCycle(issues []jiraservice.HistoryIssue, types, completed map[string]struct{}, start, end time.Time) (lead, cycle *float64, n, cycleN int) {
	var leadSum, cycleSum float64
	for _, is := range issues {
		if !inSet(types, is.Type) || !delivered(is, completed) {
			continue
		}
		if is.Resolved.Before(start) || !is.Resolved.Before(end) {
			continue
		}
		n++
		leadSum += is.Resolved.Sub(is.Created).Hours() / 24
		if t, ok := inProgressAt(is); ok && !t.After(is.Resolved) {
			cycleN++
			cycleSum += is.Resolved.Sub(t).Hours() / 24
		}
	}
	if n > 0 {
		lead = ptr(round1(leadSum / float64(n)))
	}
	if cycleN > 0 {
		cycle = ptr(round1(cycleSum / float64(cycleN)))
	}
	return lead, cycle, n, cycleN
}

func countOpen(bugs []jiraservice.HistoryIssue, at time.Time) int {
	n := 0
	for _, b := range bugs {
		if openAt(b, at) {
			n++
		}
	}
	return n
}

// loggedFixed counts bugs created in [start, end) and bugs resolved in it.
func loggedFixed(bugs []jiraservice.HistoryIssue, completed map[string]struct{}, start, end time.Time) (logged, fixed int) {
	for _, b := range bugs {
		if !b.Created.Before(start) && b.Created.Before(end) {
			logged++
		}
		if delivered(b, completed) && !b.Resolved.Before(start) && b.Resolved.Before(end) {
			fixed++
		}
	}
	return logged, fixed
}

func openByPriority(bugs []jiraservice.HistoryIssue, at time.Time) map[string]int {
	counts := map[string]int{}
	for _, b := range bugs {
		if openAt(b, at) {
			counts[priorityName(b.Priority)]++
		}
	}
	return counts
}

func priorityName(p string) string {
	if strings.TrimSpace(p) == "" {
		return "None"
	}
	return p
}

// severityOrder ranks well-known priority names from most to least severe.
var severityOrder = []string{
	"Blocker", "P0", "Critical", "Highest", "Very High", "Urgent", "Prio 1", "P1",
	"High", "Major", "Prio 2", "P2", "Medium", "Normal", "Average", "Prio 3", "P3",
	"Low", "Minor", "Prio 4", "P4", "Lowest", "Trivial", "Prio 5", "Unprioritized", "Undefined", "None",
}

// sortPriorities orders names by severity; unknown names follow alphabetically.
func sortPriorities(names []string) []string {
	rank := map[string]int{}
	for i, n := range severityOrder {
		rank[strings.ToLower(n)] = i
	}
	out := append([]string(nil), names...)
	sort.SliceStable(out, func(i, j int) bool {
		ri, iok := rank[strings.ToLower(out[i])]
		rj, jok := rank[strings.ToLower(out[j])]
		switch {
		case iok && jok:
			return ri < rj
		case iok != jok:
			return iok
		default:
			return out[i] < out[j]
		}
	})
	return out
}

// endOfDay is the last instant of the calendar day containing t, in t's location.
func endOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 23, 59, 59, int(time.Second-time.Nanosecond), t.Location())
}
