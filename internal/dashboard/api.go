package dashboard

import "time"

// TeamResponse represents one configured team for the frontend.
type TeamResponse struct {
	ComponentName string `json:"componentName"`
	BoardName     string `json:"boardName"`
}

// SprintResponse represents a Jira sprint for the frontend.
type SprintResponse struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	State     string     `json:"state"`
	StartDate *time.Time `json:"startDate,omitempty"`
	EndDate   *time.Time `json:"endDate,omitempty"`
}

// DashboardIssue is a single issue row in the sprint table.
type DashboardIssue struct {
	Key         string   `json:"key"`
	Summary     string   `json:"summary"`
	Epic        string   `json:"epic"`
	StoryPoints float64  `json:"storyPoints"`
	Type        string   `json:"type"`
	Status      string   `json:"status"`
	URL         string   `json:"url"`
	ActiveLabels []string `json:"activeLabels"`
}

// SprintIssuesResponse is the envelope returned by GET /api/sprints/{id}/issues.
type SprintIssuesResponse struct {
	ConfiguredLabels []string         `json:"configuredLabels"`
	Issues           []DashboardIssue `json:"issues"`
}

// LabelUpdateRequest is the body for POST /api/issues/{key}/labels.
type LabelUpdateRequest struct {
	Action string `json:"action"` // "add" or "remove"
	Label  string `json:"label"`
}
