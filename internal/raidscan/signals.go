package raidscan

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"go-word-create/internal/jiraservice"
)

// rules turns one open issue into its RAID signals.
type rules struct {
	opt              Options
	now              time.Time
	blockedStatuses  map[string]struct{}
	blockedLabels    map[string]struct{}
	critical         map[string]struct{}
	flagged          map[string]struct{} // keys of issues flagged as impediments
	inSprint         map[string]struct{} // keys of issues in the active sprint
	activeSprintID   int
	sprintElapsedPct float64 // -1 when no sprint is active
}

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

func days(d time.Duration) float64 { return math.Round(d.Hours()/24*10) / 10 }

func wholeDays(d time.Duration) string {
	n := int(d.Hours() / 24)
	if n < 1 {
		return "<1d"
	}
	return fmt.Sprintf("%dd", n)
}

func ptr(v float64) *float64 { return &v }

// lastStatusChange is when the issue last changed status; false if it never did.
func lastStatusChange(is jiraservice.RiskIssue) (time.Time, bool) {
	for i := len(is.Changes) - 1; i >= 0; i-- {
		if is.Changes[i].Field == "status" {
			return is.Changes[i].At, true
		}
	}
	return time.Time{}, false
}

// workStart is when work began: the first move to "In Progress", else the first
// status change at all, else creation.
func workStart(is jiraservice.RiskIssue) time.Time {
	var first time.Time
	for _, c := range is.Changes {
		if c.Field != "status" {
			continue
		}
		if first.IsZero() {
			first = c.At
		}
		if strings.EqualFold(strings.TrimSpace(c.ToString), "In Progress") {
			return c.At
		}
	}
	if !first.IsZero() {
		return first
	}
	return is.Created
}

func tokens(s string) map[string]struct{} { return lowerSet(strings.Fields(s)) }

// labelAddedAt is when one of the labels was last added to the issue.
func labelAddedAt(is jiraservice.RiskIssue, labels map[string]struct{}) time.Time {
	for i := len(is.Changes) - 1; i >= 0; i-- {
		c := is.Changes[i]
		if c.Field != "labels" {
			continue
		}
		before := tokens(c.FromString)
		for l := range tokens(c.ToString) {
			if _, was := before[l]; !was {
				if _, want := labels[l]; want {
					return c.At
				}
			}
		}
	}
	return time.Time{}
}

// flaggedAt is when the issue was last flagged as an impediment.
func flaggedAt(is jiraservice.RiskIssue) time.Time {
	for i := len(is.Changes) - 1; i >= 0; i-- {
		c := is.Changes[i]
		if strings.EqualFold(c.Field, "flagged") && strings.TrimSpace(c.ToString) != "" {
			return c.At
		}
	}
	return time.Time{}
}

// sprintsSeen counts the distinct sprints the issue has been part of, as far as
// the Sprint field history shows; the active sprint counts if it is in it now.
func sprintsSeen(is jiraservice.RiskIssue, activeID int, inActive bool) int {
	seen := map[string]struct{}{}
	for _, c := range is.Changes {
		if c.Field != "Sprint" {
			continue
		}
		for _, v := range []string{c.From, c.To} {
			for _, id := range strings.Split(v, ",") {
				if id = strings.TrimSpace(id); id != "" {
					seen[id] = struct{}{}
				}
			}
		}
	}
	if inActive && activeID != 0 {
		seen[fmt.Sprint(activeID)] = struct{}{}
	}
	return len(seen)
}

// blockedBy reports why the issue counts as blocked and since when (zero when unknown).
func (r *rules) blockedBy(is jiraservice.RiskIssue) (reason string, since time.Time, ok bool) {
	if inSet(r.blockedStatuses, is.Status) {
		since, _ = lastStatusChange(is)
		return fmt.Sprintf("status %q", is.Status), since, true
	}
	if _, f := r.flagged[is.Key]; f {
		return "flagged as an impediment", flaggedAt(is), true
	}
	for _, l := range is.Labels {
		if inSet(r.blockedLabels, l) {
			return fmt.Sprintf("label %q", l), labelAddedAt(is, r.blockedLabels), true
		}
	}
	return "", time.Time{}, false
}

// openBlockers lists the keys of unfinished issues this one waits for.
func openBlockers(is jiraservice.RiskIssue) []string {
	var keys []string
	for _, l := range is.Links {
		rel := strings.ToLower(l.Relation)
		if !strings.Contains(rel, "blocked by") && !strings.Contains(rel, "depends on") {
			continue
		}
		if strings.ToLower(l.StatusCategory) == "done" {
			continue
		}
		keys = append(keys, l.Key)
	}
	return keys
}

// evaluate computes the item and all signals of one open issue.
func (r *rules) evaluate(is jiraservice.RiskIssue) Item {
	_, inSprint := r.inSprint[is.Key]
	cat := strings.ToLower(is.StatusCategory)
	inProgress, notStarted := cat == "indeterminate", cat == "new"

	it := Item{
		Key: is.Key, Summary: is.Summary, Type: is.Type, Status: is.Status, StatusCategory: cat,
		Priority: is.Priority, Assignee: is.Assignee, StoryPoints: is.StoryPoints, InSprint: inSprint,
	}
	sinceStatus := is.Created
	if t, ok := lastStatusChange(is); ok {
		sinceStatus = t
	}
	if !sinceStatus.IsZero() {
		it.StatusDays = ptr(days(r.now.Sub(sinceStatus)))
	}
	var age time.Duration
	hasAge := false
	switch {
	case inProgress && !workStart(is).IsZero():
		age, hasAge = r.now.Sub(workStart(is)), true
	case notStarted && !is.Created.IsZero():
		age, hasAge = r.now.Sub(is.Created), true
	}
	if hasAge {
		it.AgeDays = ptr(days(age))
	}

	add := func(code, category, sev, label, detail string) {
		it.Signals = append(it.Signals, Signal{Code: code, Category: category, Severity: sev, Label: label, Detail: detail})
	}

	if reason, since, ok := r.blockedBy(is); ok {
		label, detail := "Blocked", fmt.Sprintf("Blocked: %s.", reason)
		if !since.IsZero() {
			label = "Blocked " + wholeDays(r.now.Sub(since))
			detail = fmt.Sprintf("Blocked for %s: %s.", wholeDays(r.now.Sub(since)), reason)
		}
		add(SigBlocked, CatIssue, SevHigh, label, detail)
	}
	if strings.EqualFold(is.Type, "Bug") && inSet(r.critical, is.Priority) {
		add(SigCritical, CatIssue, SevHigh, is.Priority+" bug", fmt.Sprintf("Open bug with priority %s.", is.Priority))
	}
	if keys := openBlockers(is); len(keys) > 0 {
		label := "Waits for " + keys[0]
		if len(keys) > 1 {
			label = fmt.Sprintf("Waits for %d issues", len(keys))
		}
		add(SigDependency, CatDependency, SevMedium, label, "Depends on unfinished work: "+strings.Join(keys, ", ")+".")
	}
	if inProgress && hasAge && days(age) >= float64(r.opt.AgingDays) {
		sev := SevMedium
		if days(age) >= float64(2*r.opt.AgingDays) {
			sev = SevHigh
		}
		add(SigAging, CatRisk, sev, "In work "+wholeDays(age),
			fmt.Sprintf("In progress for %s; the limit is %d days (high at %d).", wholeDays(age), r.opt.AgingDays, 2*r.opt.AgingDays))
	}
	if inProgress && !is.Updated.IsZero() && days(r.now.Sub(is.Updated)) >= float64(r.opt.StaleDays) {
		add(SigStale, CatRisk, SevMedium, "No update "+wholeDays(r.now.Sub(is.Updated)),
			fmt.Sprintf("In progress but not updated for %s; the limit is %d days.", wholeDays(r.now.Sub(is.Updated)), r.opt.StaleDays))
	}
	if notStarted && inSprint && r.sprintElapsedPct >= float64(r.opt.LateStartPct) {
		add(SigLateStart, CatRisk, SevMedium, "Not started",
			fmt.Sprintf("Still to do with %.0f%% of the sprint elapsed; the limit is %d%%.", r.sprintElapsedPct, r.opt.LateStartPct))
	}
	// Backlog items that once bounced through sprints are not "rolling over"; only work in a sprint or under way is.
	if n := sprintsSeen(is, r.activeSprintID, inSprint); (inSprint || inProgress) && n >= r.opt.RolloverSprints {
		add(SigRolledOver, CatRisk, SevMedium, fmt.Sprintf("In %d sprints", n),
			fmt.Sprintf("Has been part of %d sprints without finishing; the limit is %d.", n, r.opt.RolloverSprints))
	}
	if !is.DueDate.IsZero() && is.DueDate.Before(startOfDay(r.now)) {
		late := wholeDays(r.now.Sub(is.DueDate))
		add(SigOverdue, CatRisk, SevHigh, "Overdue "+late, fmt.Sprintf("Due %s, %s ago.", is.DueDate.Format("2006-01-02"), late))
	}
	if inSprint && !strings.EqualFold(is.Type, "Bug") && is.StoryPoints <= 0 {
		add(SigUnestimated, CatAssumption, SevLow, "No estimate", "In the sprint without story points, so the scope is understated.")
	}
	if inProgress && strings.TrimSpace(is.Assignee) == "" {
		add(SigUnassigned, CatAssumption, SevLow, "No assignee", "In progress with nobody assigned.")
	}

	for _, s := range it.Signals {
		it.Score += severityWeight(s.Severity)
	}
	return it
}

func severityWeight(sev string) int {
	switch sev {
	case SevHigh:
		return 3
	case SevMedium:
		return 2
	default:
		return 1
	}
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// topSeverity is the weight of the item's most severe signal.
func topSeverity(it Item) int {
	top := 0
	for _, s := range it.Signals {
		if w := severityWeight(s.Severity); w > top {
			top = w
		}
	}
	return top
}

// rank orders items: most severe signal first, then highest score, then oldest work, then key.
func rank(items []Item) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if ta, tb := topSeverity(a), topSeverity(b); ta != tb {
			return ta > tb
		}
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		switch {
		case a.AgeDays != nil && b.AgeDays != nil && *a.AgeDays != *b.AgeDays:
			return *a.AgeDays > *b.AgeDays
		case (a.AgeDays != nil) != (b.AgeDays != nil):
			return a.AgeDays != nil
		}
		return a.Key < b.Key
	})
}

// summarize counts issues per category and signal.
func summarize(open int, items []Item) Summary {
	s := Summary{Open: open, Flagged: len(items), ByCategory: map[string]int{}, HighByCategory: map[string]int{}, BySignal: map[string]int{}}
	for _, c := range []string{CatRisk, CatAssumption, CatIssue, CatDependency} {
		s.ByCategory[c], s.HighByCategory[c] = 0, 0
	}
	for _, it := range items {
		cats, high, codes := map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}
		for _, sg := range it.Signals {
			cats[sg.Category] = struct{}{}
			codes[sg.Code] = struct{}{}
			if sg.Severity == SevHigh {
				high[sg.Category] = struct{}{}
			}
		}
		for c := range cats {
			s.ByCategory[c]++
		}
		for c := range high {
			s.HighByCategory[c]++
		}
		for c := range codes {
			s.BySignal[c]++
		}
	}
	return s
}

// agingBuckets groups in-progress issues by days since work started.
func agingBuckets(items []jiraservice.RiskIssue, now time.Time) []Bucket {
	bounds := []struct {
		label string
		upTo  float64 // exclusive, in days
	}{{"0–3 d", 4}, {"4–7 d", 8}, {"8–14 d", 15}, {"15–30 d", 31}, {"31+ d", math.Inf(1)}}
	out := make([]Bucket, len(bounds))
	for i, b := range bounds {
		out[i].Label = b.label
	}
	for _, is := range items {
		if strings.ToLower(is.StatusCategory) != "indeterminate" {
			continue
		}
		start := workStart(is)
		if start.IsZero() {
			continue
		}
		d := now.Sub(start).Hours() / 24
		for i, b := range bounds {
			if d < b.upTo {
				out[i].Count++
				break
			}
		}
	}
	return out
}
