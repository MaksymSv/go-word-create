package mcpserver

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"go-word-create/internal/config"
	"go-word-create/internal/issuecsv"
	"go-word-create/internal/jiraservice"
	"go-word-create/internal/sprintview"
)

const defaultSprintLimit = 5

var issueKeyRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*-[0-9]+$`)

// --- list_teams ---

type listTeamsIn struct{}

type teamInfo struct {
	ComponentName string `json:"componentName"`
	BoardName     string `json:"boardName"`
}

type listTeamsOut struct {
	Teams            []teamInfo `json:"teams"`
	ConfiguredLabels []string   `json:"configuredLabels"`
	IssueTypes       []string   `json:"issueTypes"`
}

func (h *handler) listTeams(_ context.Context, _ *mcp.CallToolRequest, _ listTeamsIn) (*mcp.CallToolResult, listTeamsOut, error) {
	out := listTeamsOut{
		Teams:            make([]teamInfo, 0, len(h.cfg.Teams)),
		ConfiguredLabels: h.cfg.ReportLabels,
		IssueTypes:       h.cfg.JiraIssueTypes,
	}
	for _, t := range h.cfg.Teams {
		out.Teams = append(out.Teams, teamInfo{ComponentName: t.ComponentName, BoardName: t.BoardName})
	}
	return nil, out, nil
}

// --- list_sprints ---

type listSprintsIn struct {
	Team  string `json:"team" jsonschema:"team component name exactly as returned by list_teams"`
	Limit int    `json:"limit,omitempty" jsonschema:"max sprints to return, newest first (default 5)"`
}

type sprintInfo struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	State     string     `json:"state"`
	StartDate *time.Time `json:"startDate,omitempty"`
	EndDate   *time.Time `json:"endDate,omitempty"`
}

type listSprintsOut struct {
	Team    string       `json:"team"`
	Board   string       `json:"board"`
	Sprints []sprintInfo `json:"sprints"`
}

func (h *handler) findTeam(component string) (config.TeamEntry, error) {
	for _, t := range h.cfg.Teams {
		if t.ComponentName == component {
			return t, nil
		}
	}
	names := make([]string, 0, len(h.cfg.Teams))
	for _, t := range h.cfg.Teams {
		names = append(names, t.ComponentName)
	}
	return config.TeamEntry{}, fmt.Errorf("team %q not found (names are case-sensitive); configured teams: %s", component, strings.Join(names, ", "))
}

func (h *handler) listSprints(_ context.Context, _ *mcp.CallToolRequest, in listSprintsIn) (*mcp.CallToolResult, listSprintsOut, error) {
	team, err := h.findTeam(in.Team)
	if err != nil {
		return nil, listSprintsOut{}, err
	}
	limit := in.Limit
	if limit <= 0 {
		limit = defaultSprintLimit
	}
	sprints, err := h.jira.GetSprintsForBoard(team.BoardName, limit)
	if err != nil {
		return nil, listSprintsOut{}, fmt.Errorf("team %q: failed to get sprints for board %q: %w", team.ComponentName, team.BoardName, err)
	}
	out := listSprintsOut{Team: team.ComponentName, Board: team.BoardName, Sprints: make([]sprintInfo, 0, len(sprints))}
	for _, s := range sprints {
		out.Sprints = append(out.Sprints, sprintInfo{ID: s.ID, Name: s.Name, State: s.State, StartDate: s.StartDate, EndDate: s.EndDate})
	}
	return nil, out, nil
}

// --- get_sprint_issues ---

type getSprintIssuesIn struct {
	SprintID int      `json:"sprint_id" jsonschema:"numeric sprint ID from list_sprints"`
	Statuses []string `json:"statuses,omitempty" jsonschema:"only include issues in these statuses (case-insensitive); omit for all"`
}

type getSprintIssuesOut struct {
	SprintID          int                `json:"sprintId"`
	ConfiguredLabels  []string           `json:"configuredLabels"`
	AvailableStatuses []string           `json:"availableStatuses"`
	AppliedStatuses   []string           `json:"appliedStatuses,omitempty"`
	UnmatchedStatuses []string           `json:"unmatchedStatuses,omitempty"`
	Totals            Bucket             `json:"totals"`
	Unlabeled         Bucket             `json:"unlabeled"`
	LabelSummary      []LabelBucket      `json:"labelSummary"`
	Issues            []sprintview.Issue `json:"issues"`
}

func (h *handler) getSprintIssues(_ context.Context, _ *mcp.CallToolRequest, in getSprintIssuesIn) (*mcp.CallToolResult, getSprintIssuesOut, error) {
	if in.SprintID <= 0 {
		return nil, getSprintIssuesOut{}, fmt.Errorf("sprint_id must be a positive integer (use list_sprints to find it)")
	}
	view, err := sprintview.Build(h.jira, h.cfg.ProjectKey, h.cfg.ReportLabels, in.SprintID)
	if err != nil {
		return nil, getSprintIssuesOut{}, fmt.Errorf("failed to load sprint issues (sprintID=%d): %w", in.SprintID, err)
	}
	kept, unmatched := filterByStatus(view.Issues, in.Statuses, func(i sprintview.Issue) string { return i.Status })
	perLabel, total, unlabeled := summarize(kept, view.ConfiguredLabels)
	return nil, getSprintIssuesOut{
		SprintID:          in.SprintID,
		ConfiguredLabels:  view.ConfiguredLabels,
		AvailableStatuses: availableStatuses(view.Issues),
		AppliedStatuses:   in.Statuses,
		UnmatchedStatuses: unmatched,
		Totals:            total,
		Unlabeled:         unlabeled,
		LabelSummary:      perLabel,
		Issues:            kept,
	}, nil
}

// --- sprint_label_report ---

type sprintLabelReportIn struct {
	Sprints []string `json:"sprints" jsonschema:"exact, case-sensitive sprint names, e.g. [\"Sprint 16\"]"`
	Team    string   `json:"team,omitempty" jsonschema:"restrict to one team's board and component; omit for all teams"`
}

type sprintLabelReportOut struct {
	ConfiguredLabels []string            `json:"configuredLabels"`
	Reports          []SprintLabelReport `json:"reports"`
}

func (h *handler) sprintLabelReport(_ context.Context, _ *mcp.CallToolRequest, in sprintLabelReportIn) (*mcp.CallToolResult, sprintLabelReportOut, error) {
	var names []string
	for _, s := range in.Sprints {
		if t := strings.TrimSpace(s); t != "" {
			names = append(names, t)
		}
	}
	if len(names) == 0 {
		return nil, sprintLabelReportOut{}, fmt.Errorf("sprints must contain at least one sprint name")
	}

	boards := h.cfg.GetAllBoardNames()
	component := ""
	if in.Team != "" {
		team, err := h.findTeam(in.Team)
		if err != nil {
			return nil, sprintLabelReportOut{}, err
		}
		boards = []string{team.BoardName}
		component = team.ComponentName
	}

	out := sprintLabelReportOut{ConfiguredLabels: h.cfg.ReportLabels, Reports: make([]SprintLabelReport, 0, len(names))}
	for _, name := range names {
		issues, err := h.jira.GetSprintIssues(h.cfg.ProjectKey, name, boards, h.cfg.JiraIssueTypes)
		if err != nil {
			return nil, sprintLabelReportOut{}, fmt.Errorf("sprint %q: %w", name, err)
		}
		out.Reports = append(out.Reports, buildLabelReport(name, issues, h.cfg.ReportLabels, component))
	}
	return nil, out, nil
}

// --- month_issues ---

type monthIssuesIn struct {
	Month string `json:"month" jsonschema:"month as YYYY.MM, e.g. 2025.10"`
	Team  string `json:"team,omitempty" jsonschema:"restrict to one team component; omit for all teams"`
}

type monthIssue struct {
	Key         string   `json:"key"`
	Summary     string   `json:"summary"`
	Epic        string   `json:"epic"`
	Type        string   `json:"type"`
	Status      string   `json:"status"`
	StoryPoints float64  `json:"storyPoints"`
	Implementer string   `json:"implementer,omitempty"`
	URL         string   `json:"url"`
	Labels      []string `json:"labels"`
}

type monthTeam struct {
	Component   string       `json:"component"`
	ClosedCount int          `json:"closedCount"`
	OpenCount   int          `json:"openCount"`
	Closed      []monthIssue `json:"closed"`
	Open        []monthIssue `json:"open"`
}

type monthIssuesOut struct {
	Month string      `json:"month"`
	Teams []monthTeam `json:"teams"`
}

func toMonthIssue(i jiraservice.Issue) monthIssue {
	labels := i.Labels
	if labels == nil {
		labels = []string{}
	}
	return monthIssue{
		Key: i.Key, Summary: i.Summary, Epic: i.Epic, Type: i.Type, Status: i.Status,
		StoryPoints: i.StoryPoints, Implementer: i.Implementer, URL: i.URL, Labels: labels,
	}
}

func (h *handler) monthIssues(_ context.Context, _ *mcp.CallToolRequest, in monthIssuesIn) (*mcp.CallToolResult, monthIssuesOut, error) {
	start, err := time.Parse("2006.01", in.Month)
	if err != nil {
		return nil, monthIssuesOut{}, fmt.Errorf("invalid month %q: use YYYY.MM, e.g. 2025.10", in.Month)
	}
	end := start.AddDate(0, 1, 0)

	teams := h.cfg.Teams
	if in.Team != "" {
		t, err := h.findTeam(in.Team)
		if err != nil {
			return nil, monthIssuesOut{}, err
		}
		teams = []config.TeamEntry{t}
	}

	out := monthIssuesOut{Month: in.Month, Teams: make([]monthTeam, 0, len(teams))}
	for _, team := range teams {
		issues, err := h.jira.GetIssuesInProgressDuringMonth(h.cfg.ProjectKey, team.ComponentName, start, end, h.cfg.JiraIssueTypes)
		if err != nil {
			return nil, monthIssuesOut{}, fmt.Errorf("team %q, month %s: %w", team.ComponentName, in.Month, err)
		}
		mt := monthTeam{Component: team.ComponentName, Closed: []monthIssue{}, Open: []monthIssue{}}
		for _, iss := range issues {
			if iss.Status == "Closed" {
				mt.Closed = append(mt.Closed, toMonthIssue(iss))
			} else {
				mt.Open = append(mt.Open, toMonthIssue(iss))
			}
		}
		mt.ClosedCount, mt.OpenCount = len(mt.Closed), len(mt.Open)
		out.Teams = append(out.Teams, mt)
	}
	return nil, out, nil
}

// --- set_issue_label ---

type setIssueLabelIn struct {
	IssueKey string `json:"issue_key" jsonschema:"Jira issue key, e.g. PROJ-123"`
	Label    string `json:"label" jsonschema:"one of the configured report labels (see list_teams)"`
	Action   string `json:"action" jsonschema:"add or remove"`
	Confirm  bool   `json:"confirm,omitempty" jsonschema:"must be true to apply the change; set it only after the user explicitly approved"`
}

type setIssueLabelOut struct {
	Applied  bool   `json:"applied"`
	IssueKey string `json:"issueKey"`
	Label    string `json:"label"`
	Action   string `json:"action"`
	Message  string `json:"message"`
}

func (h *handler) setIssueLabel(_ context.Context, _ *mcp.CallToolRequest, in setIssueLabelIn) (*mcp.CallToolResult, setIssueLabelOut, error) {
	key := strings.TrimSpace(in.IssueKey)
	if !issueKeyRe.MatchString(key) {
		return nil, setIssueLabelOut{}, fmt.Errorf("invalid issue_key %q: expected a Jira key like PROJ-123", in.IssueKey)
	}
	action := strings.ToLower(strings.TrimSpace(in.Action))
	if action != "add" && action != "remove" {
		return nil, setIssueLabelOut{}, fmt.Errorf(`action must be "add" or "remove", got %q`, in.Action)
	}
	allowed := false
	for _, l := range h.cfg.ReportLabels {
		if l == in.Label {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, setIssueLabelOut{}, fmt.Errorf("label %q is not in the configured label list: %s", in.Label, strings.Join(h.cfg.ReportLabels, ", "))
	}

	out := setIssueLabelOut{IssueKey: key, Label: in.Label, Action: action}
	if !in.Confirm {
		out.Message = fmt.Sprintf("PREVIEW ONLY, nothing changed. Would %s label %q on %s. Ask the user to approve, then call again with confirm=true.", action, in.Label, key)
		return nil, out, nil
	}
	if err := h.jira.UpdateIssueLabel(key, in.Label, action == "add"); err != nil {
		return nil, setIssueLabelOut{}, fmt.Errorf("issue %s: %w", key, err)
	}
	out.Applied = true
	out.Message = fmt.Sprintf("Applied: %s label %q on %s.", action, in.Label, key)
	return nil, out, nil
}

// --- sprint_issues ---

const (
	formatJSON = "json"
	formatCSV  = "csv"
)

type sprintIssuesIn struct {
	Sprints  []string `json:"sprints" jsonschema:"exact, case-sensitive sprint names, e.g. [\"ST26.20\",\"PC26.20\"]"`
	Team     string   `json:"team,omitempty" jsonschema:"restrict to one team's board and component; omit for all teams"`
	Statuses []string `json:"statuses,omitempty" jsonschema:"only include issues in these statuses (case-insensitive); omit for all"`
	Format   string   `json:"format,omitempty" jsonschema:"json (default) returns issue rows; csv additionally returns the CSV text and a suggested file name per sprint"`
}

type sprintIssueRow struct {
	Type        string  `json:"type"`
	Key         string  `json:"key"`
	Summary     string  `json:"summary"`
	Epic        string  `json:"epic"`
	StoryPoints float64 `json:"storyPoints"`
	Status      string  `json:"status"`
	URL         string  `json:"url"`
}

type sprintIssuesResult struct {
	Sprint            string           `json:"sprint"`
	Count             int              `json:"count"`
	StoryPoints       float64          `json:"storyPoints"`
	AvailableStatuses []string         `json:"availableStatuses"`
	UnmatchedStatuses []string         `json:"unmatchedStatuses,omitempty"`
	Issues            []sprintIssueRow `json:"issues"`
	FileName          string           `json:"fileName,omitempty"`
	CSV               string           `json:"csv,omitempty"`
}

type sprintIssuesOut struct {
	Sprints []sprintIssuesResult `json:"sprints"`
}

func hasComponent(i jiraservice.Issue, component string) bool {
	for _, c := range i.Components {
		if c == component {
			return true
		}
	}
	return false
}

func (h *handler) sprintIssues(_ context.Context, _ *mcp.CallToolRequest, in sprintIssuesIn) (*mcp.CallToolResult, sprintIssuesOut, error) {
	var names []string
	for _, s := range in.Sprints {
		for _, part := range strings.Split(s, ",") {
			if t := strings.TrimSpace(part); t != "" {
				names = append(names, t)
			}
		}
	}
	if len(names) == 0 {
		return nil, sprintIssuesOut{}, fmt.Errorf("sprints must contain at least one sprint name")
	}
	format := strings.ToLower(strings.TrimSpace(in.Format))
	if format == "" {
		format = formatJSON
	}
	if format != formatJSON && format != formatCSV {
		return nil, sprintIssuesOut{}, fmt.Errorf(`format must be "json" or "csv", got %q`, in.Format)
	}

	boards := h.cfg.GetAllBoardNames()
	component := ""
	if in.Team != "" {
		team, err := h.findTeam(in.Team)
		if err != nil {
			return nil, sprintIssuesOut{}, err
		}
		boards = []string{team.BoardName}
		component = team.ComponentName
	}

	now := time.Now()
	out := sprintIssuesOut{Sprints: make([]sprintIssuesResult, 0, len(names))}
	for _, name := range names {
		all, err := h.jira.GetSprintIssues(h.cfg.ProjectKey, name, boards, h.cfg.JiraIssueTypes)
		if err != nil {
			return nil, sprintIssuesOut{}, fmt.Errorf("sprint %q: %w", name, err)
		}
		if component != "" {
			scoped := make([]jiraservice.Issue, 0, len(all))
			for _, iss := range all {
				if hasComponent(iss, component) {
					scoped = append(scoped, iss)
				}
			}
			all = scoped
		}

		kept, unmatched := filterByStatus(all, in.Statuses, func(i jiraservice.Issue) string { return i.Status })
		res := sprintIssuesResult{
			Sprint:            name,
			Count:             len(kept),
			AvailableStatuses: statusesOf(all),
			UnmatchedStatuses: unmatched,
			Issues:            make([]sprintIssueRow, 0, len(kept)),
		}
		for _, iss := range kept {
			res.StoryPoints += iss.StoryPoints
			res.Issues = append(res.Issues, sprintIssueRow{
				Type: iss.Type, Key: iss.Key, Summary: iss.Summary, Epic: iss.Epic,
				StoryPoints: iss.StoryPoints, Status: iss.Status, URL: iss.URL,
			})
		}
		if format == formatCSV {
			var sb strings.Builder
			if err := issuecsv.Write(&sb, kept); err != nil {
				return nil, sprintIssuesOut{}, fmt.Errorf("sprint %q: failed to build CSV: %w", name, err)
			}
			res.CSV = sb.String()
			res.FileName = issuecsv.FileName(name, now)
		}
		out.Sprints = append(out.Sprints, res)
	}
	return nil, out, nil
}
