// Package raidscan scans a team's Jira work for RAID signals (Risks,
// Assumptions, Issues, Dependencies): blocked, aging and at-risk items and the
// scope drift of sprints. It has no dependency on HTTP.
package raidscan

import "time"

// RAID categories.
const (
	CatRisk       = "R"
	CatAssumption = "A"
	CatIssue      = "I"
	CatDependency = "D"
)

// Severity levels of a signal.
const (
	SevHigh   = "high"
	SevMedium = "medium"
	SevLow    = "low"
)

// Signal codes.
const (
	SigBlocked     = "blocked"
	SigCritical    = "critical-bug"
	SigDependency  = "dependency"
	SigAging       = "aging"
	SigStale       = "stale"
	SigLateStart   = "late-start"
	SigRolledOver  = "rolled-over"
	SigOverdue     = "overdue"
	SigUnestimated = "unestimated"
	SigUnassigned  = "unassigned"
)

// Signal is one reason an issue is on the watchlist.
type Signal struct {
	Code     string `json:"code"`
	Category string `json:"category"`
	Severity string `json:"severity"`
	Label    string `json:"label"`  // short chip text
	Detail   string `json:"detail"` // full sentence for the tooltip
}

// Item is an open issue with at least one signal.
type Item struct {
	Key            string   `json:"key"`
	Summary        string   `json:"summary"`
	Type           string   `json:"type"`
	Status         string   `json:"status"`
	StatusCategory string   `json:"statusCategory"`
	Priority       string   `json:"priority"`
	Assignee       string   `json:"assignee"`
	StoryPoints    float64  `json:"storyPoints"`
	InSprint       bool     `json:"inSprint"`
	AgeDays        *float64 `json:"ageDays"` // days since work started (in progress) or since created (not started); nil when unknown
	StatusDays     *float64 `json:"statusDays"`
	Score          int      `json:"score"`
	Signals        []Signal `json:"signals"`
}

// Summary counts what the scan found. ByCategory and BySignal count issues,
// not signals; HighByCategory counts issues with a high-severity signal.
type Summary struct {
	Open           int            `json:"open"` // open issues examined
	Flagged        int            `json:"flagged"`
	ByCategory     map[string]int `json:"byCategory"`
	HighByCategory map[string]int `json:"highByCategory"`
	BySignal       map[string]int `json:"bySignal"`
}

// Bucket is one bar of the aging chart.
type Bucket struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// ScopePoint is the story-point scope of one sprint. Final = Committed -
// Removed + Added + Reestimated. DriftPct is (Final - Committed) / Committed in
// percent; nil when nothing was committed.
type ScopePoint struct {
	ID            int      `json:"id"`
	Sprint        string   `json:"sprint"`
	Active        bool     `json:"active"`
	Committed     float64  `json:"committed"`
	Added         float64  `json:"added"`
	Removed       float64  `json:"removed"`
	Reestimated   float64  `json:"reestimated"`
	Final         float64  `json:"final"`
	Completed     float64  `json:"completed"`
	CarriedOver   float64  `json:"carriedOver"`
	IssuesAdded   int      `json:"issuesAdded"`
	IssuesRemoved int      `json:"issuesRemoved"`
	IssuesCarried int      `json:"issuesCarried"`
	DriftPct      *float64 `json:"driftPct"`
}

// SprintPulse compares progress with elapsed time in the running sprint.
type SprintPulse struct {
	Name       string     `json:"name"`
	StartDate  time.Time  `json:"startDate"`
	EndDate    time.Time  `json:"endDate"`
	ElapsedPct float64    `json:"elapsedPct"`
	DaysLeft   float64    `json:"daysLeft"`
	DonePct    *float64   `json:"donePct"` // completed / current scope, nil when the scope is empty
	Scope      ScopePoint `json:"scope"`
	NotStarted int        `json:"notStarted"` // open sprint issues still in the "to do" category
	InProgress int        `json:"inProgress"`
}

// Snapshot is the watchlist size on one day. Jira cannot tell what was blocked
// last week, so the trend starts the first day the scan ran.
type Snapshot struct {
	Date    string `json:"date"`
	Open    int    `json:"open"`
	Flagged int    `json:"flagged"`
	Blocked int    `json:"blocked"`
	Aging   int    `json:"aging"`
}

// Thresholds echoes the rules the scan applied so the page can state them.
type Thresholds struct {
	BlockedStatuses    []string `json:"blockedStatuses"`
	BlockedLabels      []string `json:"blockedLabels"`
	CriticalPriorities []string `json:"criticalPriorities"`
	AgingDays          int      `json:"agingDays"`
	StaleDays          int      `json:"staleDays"`
	RolloverSprints    int      `json:"rolloverSprints"`
	LateStartPct       int      `json:"lateStartPct"`
	DriftPct           int      `json:"driftPct"`
}

// Report is everything the RAID page shows for one team. Sections that could
// not be computed are empty and explained in Errors.
type Report struct {
	Team          string            `json:"team"`
	Board         string            `json:"board"`
	GeneratedAt   time.Time         `json:"generatedAt"`
	SprintCount   int               `json:"sprintCount"`
	Thresholds    Thresholds        `json:"thresholds"`
	Summary       Summary           `json:"summary"`
	SprintSummary *Summary          `json:"sprintSummary"` // the same counts for the running sprint's issues only; nil when there is none
	Items         []Item            `json:"items"`
	AgingBuckets  []Bucket          `json:"agingBuckets"`
	Scope         []ScopePoint      `json:"scope"`
	Pulse         *SprintPulse      `json:"pulse"`
	Trend         []Snapshot        `json:"trend"`
	Warnings      []string          `json:"warnings,omitempty"`
	Errors        map[string]string `json:"errors,omitempty"`
}
