// Package deliverystats computes the delivery dashboard metrics (velocity,
// commitment rate, backlog health, lead/cycle time and bug trends) for a team
// from Jira data. It has no dependency on HTTP.
package deliverystats

import "time"

// SprintRef identifies one reported sprint. EndDate is the effective end: the
// close time when the sprint was completed, otherwise the planned end.
type SprintRef struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	StartDate time.Time `json:"startDate"`
	EndDate   time.Time `json:"endDate"`
}

// VelocityPoint is committed vs completed story points plus the trailing
// average of completed points over the last three sprints (this one included).
// Values are nil for sprints whose velocity was never seen (Jira's velocity
// chart covers only the last 7 sprints).
type VelocityPoint struct {
	Sprint    string   `json:"sprint"`
	Committed *float64 `json:"committed"`
	Completed *float64 `json:"completed"`
	AvgLast3  *float64 `json:"avgLast3"`
}

// CommitmentPoint is completed / committed in percent; nil when nothing was committed.
type CommitmentPoint struct {
	Sprint  string   `json:"sprint"`
	RatePct *float64 `json:"ratePct"`
}

// BacklogPoint is the groomable backlog (open issues outside any sprint) when the sprint started.
type BacklogPoint struct {
	Sprint        string   `json:"sprint"`
	StoryPoints   float64  `json:"storyPoints"`
	Issues        int      `json:"issues"`
	Unestimated   int      `json:"unestimated"`
	RunwaySprints *float64 `json:"runwaySprints"`
}

// LeadCyclePoint holds average lead and cycle time in days for issues completed in the sprint.
type LeadCyclePoint struct {
	Sprint      string   `json:"sprint"`
	LeadDays    *float64 `json:"leadDays"`
	CycleDays   *float64 `json:"cycleDays"`
	Issues      int      `json:"issues"`
	CycleIssues int      `json:"cycleIssues"`
}

// DayPoint is the number of open bugs at the end of a day.
type DayPoint struct {
	Date string `json:"date"`
	Open int    `json:"open"`
}

// BugDynamicsPoint counts bugs logged and fixed during a sprint.
type BugDynamicsPoint struct {
	Sprint string `json:"sprint"`
	Logged int    `json:"logged"`
	Fixed  int    `json:"fixed"`
}

// PriorityPoint is the open-bug count per priority at the end of a sprint.
type PriorityPoint struct {
	Sprint string         `json:"sprint"`
	Counts map[string]int `json:"counts"`
}

// OpenBugsByPriority lists the priorities that occur (most severe first) and one point per sprint.
type OpenBugsByPriority struct {
	Priorities []string        `json:"priorities"`
	Points     []PriorityPoint `json:"points"`
}

// Report is everything the dashboard shows for one team. Sections that could
// not be computed are nil and explained in Errors.
type Report struct {
	Team               string              `json:"team"`
	Board              string              `json:"board"`
	GeneratedAt        time.Time           `json:"generatedAt"`
	SprintCount        int                 `json:"sprintCount"`
	BugDays            int                 `json:"bugDays"`
	Sprints            []SprintRef         `json:"sprints"`
	Velocity           []VelocityPoint     `json:"velocity"`
	CommitmentRate     []CommitmentPoint   `json:"commitmentRate"`
	BacklogHealth      []BacklogPoint      `json:"backlogHealth"`
	LeadCycle          []LeadCyclePoint    `json:"leadCycle"`
	OpenBugsDaily      []DayPoint          `json:"openBugsDaily"`
	BugDynamics        []BugDynamicsPoint  `json:"bugDynamics"`
	OpenBugsByPriority *OpenBugsByPriority `json:"openBugsByPriority"`
	Errors             map[string]string   `json:"errors,omitempty"`
}
