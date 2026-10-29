package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	jira "github.com/andygrunwald/go-jira"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"go-word-create/internal/config"
	"go-word-create/internal/jiraservice"
)

type updateCall struct {
	key, label string
	add        bool
}

type fakeJira struct {
	epics        map[string]string
	epicsErr     error
	sprintIssues []jiraservice.Issue // GetIssuesFromSprintWithChangelog
	sprintErr    error
	namedIssues  map[string][]jiraservice.Issue // GetSprintIssues by sprint name
	monthIssues  map[string][]jiraservice.Issue // by component
	sprints      []jira.Sprint
	updates      []updateCall
	updateErr    error
	lastBoards   []string
}

func (f *fakeJira) LoadEpics(string) (map[string]string, error) { return f.epics, f.epicsErr }

func (f *fakeJira) GetIssuesFromSprintWithChangelog(int, map[string]string, map[string]struct{}) ([]jiraservice.Issue, error) {
	return f.sprintIssues, f.sprintErr
}

func (f *fakeJira) GetSprintsForBoard(_ string, limit int) ([]jira.Sprint, error) {
	if limit < len(f.sprints) {
		return f.sprints[:limit], nil
	}
	return f.sprints, nil
}

func (f *fakeJira) GetSprintIssues(_, name string, boards, _ []string) ([]jiraservice.Issue, error) {
	f.lastBoards = boards
	iss, ok := f.namedIssues[name]
	if !ok {
		return nil, errors.New("sprint not found")
	}
	return iss, nil
}

func (f *fakeJira) GetIssuesInProgressDuringMonth(_, component string, _, _ time.Time, _ []string) ([]jiraservice.Issue, error) {
	return f.monthIssues[component], nil
}

func (f *fakeJira) UpdateIssueLabel(key, label string, add bool) error {
	f.updates = append(f.updates, updateCall{key, label, add})
	return f.updateErr
}

func testConfig() *config.Config {
	return &config.Config{
		ProjectKey:     "PRJ",
		ReportLabels:   []string{"ai-assisted", "ai-assisted-dev"},
		JiraIssueTypes: []string{"Bug", "Story", "Task"},
		Teams: []config.TeamEntry{
			{ComponentName: "Alpha", BoardName: "Alpha Board"},
			{ComponentName: "Beta", BoardName: "Beta Board"},
		},
	}
}

func newHandler(f *fakeJira) *handler { return &handler{cfg: testConfig(), jira: f} }

func TestGetSprintIssuesStatusFilterAndSummary(t *testing.T) {
	f := &fakeJira{sprintIssues: []jiraservice.Issue{
		{Key: "PRJ-1", Status: "In Review", StoryPoints: 3, Labels: []string{"ai-assisted", "ai-assisted-dev", "other"}},
		{Key: "PRJ-2", Status: "in review", StoryPoints: 2, Labels: []string{"ai-assisted"}},
		{Key: "PRJ-3", Status: "Closed", StoryPoints: 5},
		{Key: "PRJ-4", Status: "To Do", StoryPoints: 1},
	}}
	_, out, err := newHandler(f).getSprintIssues(context.Background(), nil, getSprintIssuesIn{
		SprintID: 7, Statuses: []string{" IN REVIEW ", "Blocked"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Issues) != 2 {
		t.Fatalf("want 2 filtered issues, got %d", len(out.Issues))
	}
	if out.Totals != (Bucket{Count: 2, StoryPoints: 5}) {
		t.Errorf("totals = %+v", out.Totals)
	}
	if out.Unlabeled.Count != 0 {
		t.Errorf("unlabeled = %+v", out.Unlabeled)
	}
	want := []LabelBucket{{"ai-assisted", 2, 5}, {"ai-assisted-dev", 1, 3}}
	if !reflect.DeepEqual(out.LabelSummary, want) {
		t.Errorf("labelSummary = %+v, want %+v", out.LabelSummary, want)
	}
	if !reflect.DeepEqual(out.UnmatchedStatuses, []string{"Blocked"}) {
		t.Errorf("unmatched = %v", out.UnmatchedStatuses)
	}
	if len(out.AvailableStatuses) != 4 {
		t.Errorf("availableStatuses = %v (computed over the unfiltered set)", out.AvailableStatuses)
	}
}

func TestGetSprintIssuesNoFilterCountsUnlabeled(t *testing.T) {
	f := &fakeJira{sprintIssues: []jiraservice.Issue{
		{Key: "PRJ-1", Status: "Closed", StoryPoints: 3, Labels: []string{"ai-assisted"}},
		{Key: "PRJ-2", Status: "Closed", StoryPoints: 2},
	}}
	_, out, err := newHandler(f).getSprintIssues(context.Background(), nil, getSprintIssuesIn{SprintID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if out.Unlabeled != (Bucket{Count: 1, StoryPoints: 2}) {
		t.Errorf("unlabeled = %+v", out.Unlabeled)
	}
	if out.Totals.Count != 2 {
		t.Errorf("totals = %+v", out.Totals)
	}
}

func TestGetSprintIssuesRejectsBadID(t *testing.T) {
	if _, _, err := newHandler(&fakeJira{}).getSprintIssues(context.Background(), nil, getSprintIssuesIn{}); err == nil {
		t.Fatal("expected error for sprint_id 0")
	}
}

func TestGetSprintIssuesToleratesEpicFailure(t *testing.T) {
	f := &fakeJira{epicsErr: errors.New("boom"), sprintIssues: []jiraservice.Issue{{Key: "PRJ-1", Status: "Closed"}}}
	_, out, err := newHandler(f).getSprintIssues(context.Background(), nil, getSprintIssuesIn{SprintID: 1})
	if err != nil || len(out.Issues) != 1 {
		t.Fatalf("err=%v issues=%d", err, len(out.Issues))
	}
}

func TestSprintLabelReportMapping(t *testing.T) {
	f := &fakeJira{namedIssues: map[string][]jiraservice.Issue{"Sprint 1": {
		{Key: "PRJ-1", StoryPoints: 3, Components: []string{"Alpha"}, Labels: []string{"ai-assisted", "ai-assisted-dev"}},
		{Key: "PRJ-2", StoryPoints: 5, Components: []string{"Alpha"}},
		{Key: "PRJ-3", StoryPoints: 2},
	}}}
	_, out, err := newHandler(f).sprintLabelReport(context.Background(), nil, sprintLabelReportIn{Sprints: []string{"Sprint 1"}})
	if err != nil {
		t.Fatal(err)
	}
	comps := out.Reports[0].Components
	if len(comps) != 2 || comps[0].Component != "Alpha" || comps[1].Component != "No Component" {
		t.Fatalf("components = %+v", comps)
	}
	a := comps[0]
	if a.IssueCount != 2 || a.StoryPoints != 8 {
		t.Errorf("Alpha totals = %d/%v", a.IssueCount, a.StoryPoints)
	}
	if len(a.LabelGroups) != 2 || a.LabelGroups[0].Count != 1 || a.LabelGroups[1].Count != 1 {
		t.Errorf("multi-label issue must count in both groups: %+v", a.LabelGroups)
	}
	if a.LabelGroups[0].CountPct != 50 || a.LabelGroups[0].StoryPointsPct != 37.5 {
		t.Errorf("pct = %v / %v", a.LabelGroups[0].CountPct, a.LabelGroups[0].StoryPointsPct)
	}
	if len(a.UnlabeledIssues) != 1 || a.UnlabeledIssues[0].Key != "PRJ-2" {
		t.Errorf("unlabeled = %+v", a.UnlabeledIssues)
	}
}

func TestSprintLabelReportTeamFilterAndBoards(t *testing.T) {
	f := &fakeJira{namedIssues: map[string][]jiraservice.Issue{"S": {
		{Key: "PRJ-1", Components: []string{"Alpha"}},
		{Key: "PRJ-2", Components: []string{"Beta"}},
	}}}
	_, out, err := newHandler(f).sprintLabelReport(context.Background(), nil, sprintLabelReportIn{Sprints: []string{"S"}, Team: "Beta"})
	if err != nil {
		t.Fatal(err)
	}
	if c := out.Reports[0].Components; len(c) != 1 || c[0].Component != "Beta" {
		t.Errorf("components = %+v", c)
	}
	if !reflect.DeepEqual(f.lastBoards, []string{"Beta Board"}) {
		t.Errorf("boards = %v", f.lastBoards)
	}
}

func TestSprintLabelReportErrors(t *testing.T) {
	h := newHandler(&fakeJira{namedIssues: map[string][]jiraservice.Issue{}})
	if _, _, err := h.sprintLabelReport(context.Background(), nil, sprintLabelReportIn{Sprints: []string{" "}}); err == nil {
		t.Error("expected error for empty sprint list")
	}
	if _, _, err := h.sprintLabelReport(context.Background(), nil, sprintLabelReportIn{Sprints: []string{"x"}, Team: "Nope"}); err == nil || !strings.Contains(err.Error(), "Alpha") {
		t.Errorf("unknown team error should list teams, got %v", err)
	}
	if _, _, err := h.sprintLabelReport(context.Background(), nil, sprintLabelReportIn{Sprints: []string{"missing"}}); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Errorf("error should name the sprint, got %v", err)
	}
}

func TestMonthIssuesSplit(t *testing.T) {
	f := &fakeJira{monthIssues: map[string][]jiraservice.Issue{
		"Alpha": {{Key: "PRJ-1", Status: "Closed"}, {Key: "PRJ-2", Status: "In Progress"}, {Key: "PRJ-3", Status: "Closed"}},
		"Beta":  {},
	}}
	_, out, err := newHandler(f).monthIssues(context.Background(), nil, monthIssuesIn{Month: "2025.10"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Teams) != 2 {
		t.Fatalf("teams = %d", len(out.Teams))
	}
	if a := out.Teams[0]; a.ClosedCount != 2 || a.OpenCount != 1 {
		t.Errorf("Alpha = %+v", a)
	}
	if b := out.Teams[1]; b.ClosedCount != 0 || b.Closed == nil || b.Open == nil {
		t.Errorf("Beta must have empty non-nil slices: %+v", b)
	}

	_, one, err := newHandler(f).monthIssues(context.Background(), nil, monthIssuesIn{Month: "2025.10", Team: "Alpha"})
	if err != nil || len(one.Teams) != 1 {
		t.Fatalf("team filter: err=%v teams=%d", err, len(one.Teams))
	}
	if _, _, err := newHandler(f).monthIssues(context.Background(), nil, monthIssuesIn{Month: "2025-10"}); err == nil {
		t.Error("expected error for bad month format")
	}
}

func TestListSprintsDefaultLimit(t *testing.T) {
	var sprints []jira.Sprint
	for i := 1; i <= 8; i++ {
		sprints = append(sprints, jira.Sprint{ID: i, Name: "S"})
	}
	_, out, err := newHandler(&fakeJira{sprints: sprints}).listSprints(context.Background(), nil, listSprintsIn{Team: "Alpha"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Sprints) != defaultSprintLimit || out.Board != "Alpha Board" {
		t.Errorf("sprints=%d board=%q", len(out.Sprints), out.Board)
	}
}

func TestSetIssueLabel(t *testing.T) {
	ctx := context.Background()

	t.Run("preview does not call Jira", func(t *testing.T) {
		f := &fakeJira{}
		_, out, err := newHandler(f).setIssueLabel(ctx, nil, setIssueLabelIn{IssueKey: "PRJ-1", Label: "ai-assisted", Action: "add"})
		if err != nil {
			t.Fatal(err)
		}
		if out.Applied || len(f.updates) != 0 || !strings.Contains(out.Message, "PREVIEW") {
			t.Errorf("out=%+v updates=%v", out, f.updates)
		}
	})

	t.Run("confirm applies", func(t *testing.T) {
		f := &fakeJira{}
		_, out, err := newHandler(f).setIssueLabel(ctx, nil, setIssueLabelIn{IssueKey: "PRJ-1", Label: "ai-assisted", Action: "Remove", Confirm: true})
		if err != nil {
			t.Fatal(err)
		}
		if !out.Applied || !reflect.DeepEqual(f.updates, []updateCall{{"PRJ-1", "ai-assisted", false}}) {
			t.Errorf("out=%+v updates=%v", out, f.updates)
		}
	})

	t.Run("rejects invalid input without calling Jira", func(t *testing.T) {
		cases := []setIssueLabelIn{
			{IssueKey: "PRJ-1", Label: "not-configured", Action: "add", Confirm: true},
			{IssueKey: "bad key", Label: "ai-assisted", Action: "add", Confirm: true},
			{IssueKey: "PRJ-1", Label: "ai-assisted", Action: "toggle", Confirm: true},
		}
		for _, in := range cases {
			f := &fakeJira{}
			if _, _, err := newHandler(f).setIssueLabel(ctx, nil, in); err == nil {
				t.Errorf("%+v: expected error", in)
			}
			if len(f.updates) != 0 {
				t.Errorf("%+v: Jira must not be called", in)
			}
		}
	})

	t.Run("Jira failure surfaces with issue key", func(t *testing.T) {
		f := &fakeJira{updateErr: errors.New("403")}
		_, _, err := newHandler(f).setIssueLabel(ctx, nil, setIssueLabelIn{IssueKey: "PRJ-9", Label: "ai-assisted", Action: "add", Confirm: true})
		if err == nil || !strings.Contains(err.Error(), "PRJ-9") {
			t.Errorf("err = %v", err)
		}
	})
}

func connect(t *testing.T, f *fakeJira) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	srv := New(testConfig(), f)
	ss, err := srv.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func TestServerEndToEnd(t *testing.T) {
	f := &fakeJira{}
	cs := connect(t, f)
	ctx := context.Background()

	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]*mcp.Tool{}
	for _, tl := range tools.Tools {
		got[tl.Name] = tl
	}
	for _, name := range []string{"list_teams", "list_sprints", "get_sprint_issues", "sprint_label_report", "month_issues", "set_issue_label"} {
		if got[name] == nil {
			t.Errorf("tool %q not registered", name)
		}
	}
	if w := got["set_issue_label"]; w != nil && w.Annotations.ReadOnlyHint {
		t.Error("set_issue_label must not be read-only")
	}
	if r := got["get_sprint_issues"]; r != nil && !r.Annotations.ReadOnlyHint {
		t.Error("get_sprint_issues must be read-only")
	}

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "list_teams"})
	if err != nil || res.IsError {
		t.Fatalf("list_teams: err=%v res=%+v", err, res)
	}
	b, _ := json.Marshal(res.StructuredContent)
	var lt listTeamsOut
	if err := json.Unmarshal(b, &lt); err != nil || len(lt.Teams) != 2 {
		t.Errorf("list_teams structured content = %s", b)
	}

	res, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "set_issue_label", Arguments: map[string]any{
		"issue_key": "PRJ-1", "label": "nope", "action": "add", "confirm": true,
	}})
	if err != nil {
		t.Fatalf("protocol error: %v", err)
	}
	if !res.IsError {
		t.Error("handler error must be surfaced as a tool error result")
	}
	if len(f.updates) != 0 {
		t.Error("Jira must not be called for an invalid label")
	}

	res, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "set_issue_label", Arguments: map[string]any{
		"issue_key": "PRJ-1", "label": "ai-assisted", "action": "add",
	}})
	if err != nil || res.IsError || len(f.updates) != 0 {
		t.Errorf("default confirm=false must be a no-op preview: err=%v res=%+v updates=%v", err, res, f.updates)
	}
}
