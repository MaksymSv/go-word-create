// Package mcpserver exposes the Jira sprint/label reporting functionality as
// Model Context Protocol tools so LLM clients can query it with typed JSON.
package mcpserver

import (
	"time"

	jira "github.com/andygrunwald/go-jira"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"go-word-create/internal/config"
	"go-word-create/internal/jiraservice"
	"go-word-create/internal/sprintview"
)

const (
	serverName    = "jira-reports"
	serverVersion = "1.0.0"
)

const instructions = `Read-only Jira reporting for sprints and AI-usage labels, plus one guarded write tool.
Typical flow: list_teams -> list_sprints(team) -> get_sprint_issues(sprint_id, statuses?).
sprint_label_report takes sprint NAMES (exact, case-sensitive) and covers every configured team.
set_issue_label changes Jira data: call it first with confirm=false, show the user the preview,
and only call it again with confirm=true after the user explicitly approves.`

// Jira is the subset of jiraservice.JiraService used by the tools.
type Jira interface {
	sprintview.IssueSource
	GetSprintsForBoard(boardName string, limit int) ([]jira.Sprint, error)
	GetSprintIssues(projectKey, sprintName string, boardNames, issueTypes []string) ([]jiraservice.Issue, error)
	GetIssuesInProgressDuringMonth(projectKey, component string, monthStart, monthEnd time.Time, issueTypes []string) ([]jiraservice.Issue, error)
	UpdateIssueLabel(issueKey, label string, add bool) error
}

type handler struct {
	cfg  *config.Config
	jira Jira
}

func ptr[T any](v T) *T { return &v }

// New builds an MCP server with all Jira reporting tools registered.
func New(cfg *config.Config, j Jira) *mcp.Server {
	h := &handler{cfg: cfg, jira: j}
	srv := mcp.NewServer(
		&mcp.Implementation{Name: serverName, Title: "Jira sprint & label reports", Version: serverVersion},
		&mcp.ServerOptions{Instructions: instructions},
	)

	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: ptr(true)}

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_teams",
		Description: "List configured teams (Jira component + board name), the tracked report labels, and the issue types included in reports.",
		Annotations: readOnly,
	}, h.listTeams)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_sprints",
		Description: "List the most recent active/closed sprints (newest first) of a team's board. Returns sprint IDs for get_sprint_issues.",
		Annotations: readOnly,
	}, h.listSprints)

	mcp.AddTool(srv, &mcp.Tool{
		Name: "get_sprint_issues",
		Description: "Issues of one sprint (key, summary, epic, implementer, story points, type, status, active report labels) " +
			"with per-label counts/story points. Optionally filter by status; summaries cover the filtered set.",
		Annotations: readOnly,
	}, h.getSprintIssues)

	mcp.AddTool(srv, &mcp.Tool{
		Name: "sprint_label_report",
		Description: "Label (AI-usage) report for one or more sprints by exact name, grouped by Jira component: per configured label " +
			"count, story points and percentages. Labels are non-exclusive; percentages are relative to the component total.",
		Annotations: readOnly,
	}, h.sprintLabelReport)

	mcp.AddTool(srv, &mcp.Tool{
		Name: "month_issues",
		Description: "Issues that were 'In Progress' at any time during a month (from the changelog), per team, split into " +
			"Closed (current status 'Closed') and Open.",
		Annotations: readOnly,
	}, h.monthIssues)

	mcp.AddTool(srv, &mcp.Tool{
		Name: "set_issue_label",
		Description: "WRITE: add or remove one tracked report label on a Jira issue. Without confirm=true nothing is changed and a preview " +
			"is returned. Only call with confirm=true after the user explicitly approved this exact change.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: ptr(false),
			IdempotentHint:  true,
			OpenWorldHint:   ptr(true),
		},
	}, h.setIssueLabel)

	return srv
}
