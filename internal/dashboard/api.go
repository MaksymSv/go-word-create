package dashboard

import (
	"time"

	"go-word-create/internal/sprintview"
)

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
type DashboardIssue = sprintview.Issue

// SprintIssuesResponse is the envelope returned by GET /api/sprints/{id}/issues.
type SprintIssuesResponse = sprintview.Response

// LabelUpdateRequest is the body for POST /api/issues/{key}/labels.
type LabelUpdateRequest struct {
	Action string `json:"action"` // "add" or "remove"
	Label  string `json:"label"`
}
