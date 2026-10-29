// Package sprintview builds the per-sprint issue view (issues plus which of the
// configured report labels each one carries) shared by the web dashboard and the
// MCP server.
package sprintview

import (
	"log"

	"go-word-create/internal/jiraservice"
)

// Issue is a single issue row in the sprint view.
type Issue struct {
	Key          string   `json:"key"`
	Summary      string   `json:"summary"`
	Epic         string   `json:"epic"`
	Implementer  string   `json:"implementer"`
	StoryPoints  float64  `json:"storyPoints"`
	Type         string   `json:"type"`
	Status       string   `json:"status"`
	URL          string   `json:"url"`
	ActiveLabels []string `json:"activeLabels"`
}

// Response is the envelope returned for one sprint.
type Response struct {
	ConfiguredLabels []string `json:"configuredLabels"`
	Issues           []Issue  `json:"issues"`
}

// IssueSource is the subset of jiraservice.JiraService that Build needs.
type IssueSource interface {
	LoadEpics(projectKey string) (map[string]string, error)
	GetIssuesFromSprintWithChangelog(sprintID int, epicNames map[string]string, typeFilter map[string]struct{}) ([]jiraservice.Issue, error)
}

// Build loads the issues of a sprint and marks which configured labels each carries.
// A failure to load epics is logged and tolerated (epic names stay empty).
func Build(src IssueSource, projectKey string, reportLabels []string, sprintID int) (Response, error) {
	epicNames, err := src.LoadEpics(projectKey)
	if err != nil {
		log.Printf("sprintview: failed to load epics: %v", err)
		epicNames = make(map[string]string)
	}

	issues, err := src.GetIssuesFromSprintWithChangelog(sprintID, epicNames, nil)
	if err != nil {
		return Response{}, err
	}

	configured := make(map[string]struct{}, len(reportLabels))
	for _, l := range reportLabels {
		configured[l] = struct{}{}
	}

	out := make([]Issue, 0, len(issues))
	for _, issue := range issues {
		active := []string{}
		for _, l := range issue.Labels {
			if _, ok := configured[l]; ok {
				active = append(active, l)
			}
		}
		out = append(out, Issue{
			Key:          issue.Key,
			Summary:      issue.Summary,
			Epic:         issue.Epic,
			Implementer:  issue.Implementer,
			StoryPoints:  issue.StoryPoints,
			Type:         issue.Type,
			Status:       issue.Status,
			URL:          issue.URL,
			ActiveLabels: active,
		})
	}

	return Response{ConfiguredLabels: reportLabels, Issues: out}, nil
}
