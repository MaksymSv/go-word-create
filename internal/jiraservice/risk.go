package jiraservice

import (
	"fmt"
	"sort"
	"strings"
	"time"

	jira "github.com/andygrunwald/go-jira"
)

// IssueLink is one link of an issue, seen from that issue: Relation reads as a
// sentence ("is blocked by", "blocks") and Key is the issue on the other end.
type IssueLink struct {
	Relation       string
	Key            string
	Status         string
	StatusCategory string // "done", "new" or "indeterminate"; empty when unknown
}

// RiskIssue is the view of an issue used by RAID scans: the history fields
// plus the people, labels, dates and links that risk signals are built from.
type RiskIssue struct {
	HistoryIssue
	Summary  string
	Assignee string
	Labels   []string
	Updated  time.Time
	DueDate  time.Time
	Links    []IssueLink
}

// SearchRiskIssues runs the JQL and returns every match (all pages) as
// RiskIssue. With withChangelog the field transitions are included.
func (s *JiraService) SearchRiskIssues(jql string, withChangelog bool) ([]RiskIssue, error) {
	opts := &jira.SearchOptions{
		MaxResults: 100,
		Fields: []string{
			"issuetype", "status", "priority", "resolution", "resolutiondate", "created", s.spField,
			"summary", "assignee", "labels", "updated", "duedate", "issuelinks",
		},
	}
	if withChangelog {
		opts.Expand = "changelog"
	}
	var out []RiskIssue
	err := s.client.Issue.SearchPages(jql, opts, func(is jira.Issue) error {
		out = append(out, s.toRiskIssue(is))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search issues (jql=%q): %w", jql, err)
	}
	return out, nil
}

// SearchKeys returns the keys of every issue matching the JQL.
func (s *JiraService) SearchKeys(jql string) (map[string]struct{}, error) {
	keys := map[string]struct{}{}
	err := s.client.Issue.SearchPages(jql, &jira.SearchOptions{MaxResults: 100, Fields: []string{"key"}}, func(is jira.Issue) error {
		keys[is.Key] = struct{}{}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search issue keys (jql=%q): %w", jql, err)
	}
	return keys, nil
}

func (s *JiraService) toRiskIssue(is jira.Issue) RiskIssue {
	r := RiskIssue{
		HistoryIssue: s.toHistoryIssue(is),
		Summary:      is.Fields.Summary,
		Labels:       is.Fields.Labels,
		Updated:      time.Time(is.Fields.Updated),
		DueDate:      time.Time(is.Fields.Duedate),
	}
	if is.Fields.Assignee != nil {
		r.Assignee = is.Fields.Assignee.DisplayName
	}
	for _, l := range is.Fields.IssueLinks {
		if l == nil {
			continue
		}
		other, relation := l.OutwardIssue, l.Type.Outward
		if l.InwardIssue != nil {
			other, relation = l.InwardIssue, l.Type.Inward
		}
		if other == nil {
			continue
		}
		link := IssueLink{Relation: relation, Key: other.Key}
		if other.Fields != nil && other.Fields.Status != nil {
			link.Status = other.Fields.Status.Name
			link.StatusCategory = strings.ToLower(other.Fields.Status.StatusCategory.Key)
		}
		r.Links = append(r.Links, link)
	}
	return r
}

// ActiveSprint returns the active sprint of the named board (the most recently
// started one when several are active), or nil when none is running.
func (s *JiraService) ActiveSprint(boardName string) (*SprintInfo, error) {
	all, err := s.GetSprintsForBoard(boardName, 0)
	if err != nil {
		return nil, err
	}
	var active []SprintInfo
	for _, sp := range all {
		if !strings.EqualFold(sp.State, "active") || sp.StartDate == nil {
			continue
		}
		info := SprintInfo{ID: sp.ID, Name: sp.Name, State: sp.State, Start: *sp.StartDate}
		if sp.EndDate != nil {
			info.End = *sp.EndDate
		}
		active = append(active, info)
	}
	if len(active) == 0 {
		return nil, nil
	}
	sort.SliceStable(active, func(i, j int) bool { return active[i].Start.After(active[j].Start) })
	return &active[0], nil
}

// SprintScope is the story-point scope of one sprint as Jira's sprint report
// shows it. Final = Committed - Removed + Added + Reestimated.
type SprintScope struct {
	SprintID      int
	Committed     float64 // estimates of the issues in the sprint when it started (equals the velocity chart's committed value)
	Added         float64 // estimates, when added, of issues added after the start and still in the sprint
	Removed       float64 // estimates of issues taken out after the start (they were committed)
	Reestimated   float64 // net estimate change of issues that stayed in the sprint
	Final         float64 // scope now (or at close): current estimates of the issues still in the sprint
	Completed     float64 // current estimates of issues completed in the sprint
	CarriedOver   float64 // current estimates of issues not completed in the sprint
	IssuesAdded   int
	IssuesRemoved int
	IssuesDone    int
	IssuesCarried int
}

// reportIssue is one entry of the sprint report. estimateStatistic is the
// estimate when the sprint started (or when the issue was added);
// currentEstimateStatistic is the estimate now.
type reportIssue struct {
	Key               string         `json:"key"`
	EstimateStatistic *reportEstStat `json:"estimateStatistic"`
	CurrentEstimate   *reportEstStat `json:"currentEstimateStatistic"`
}

type reportEstStat struct {
	StatFieldValue struct {
		Value float64 `json:"value"`
	} `json:"statFieldValue"`
}

func (r reportIssue) initial() float64 {
	if r.EstimateStatistic == nil {
		return 0
	}
	return r.EstimateStatistic.StatFieldValue.Value
}

func (r reportIssue) current() float64 {
	if r.CurrentEstimate == nil {
		return r.initial()
	}
	return r.CurrentEstimate.StatFieldValue.Value
}

// SprintScope reads Jira's sprint report for the sprint and reduces it to scope
// numbers. It uses the Jira Software "greenhopper" chart endpoint, which is not
// part of the documented REST API.
func (s *JiraService) SprintScope(boardName string, sprintID int) (*SprintScope, error) {
	board, err := s.GetBoard(boardName)
	if err != nil {
		return nil, err
	}
	req, err := s.client.NewRequest("GET", fmt.Sprintf("rest/greenhopper/1.0/rapid/charts/sprintreport?rapidViewId=%d&sprintId=%d", board.ID, sprintID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build sprint report request for sprint %d of board %q: %w", sprintID, boardName, err)
	}
	var payload struct {
		Contents struct {
			Completed      []reportIssue   `json:"completedIssues"`
			NotCompleted   []reportIssue   `json:"issuesNotCompletedInCurrentSprint"`
			Punted         []reportIssue   `json:"puntedIssues"`
			CompletedOther []reportIssue   `json:"issuesCompletedInAnotherSprint"`
			Added          map[string]bool `json:"issueKeysAddedDuringSprint"`
		} `json:"contents"`
	}
	if _, err := s.client.Do(req, &payload); err != nil {
		return nil, fmt.Errorf("failed to read sprint report of sprint %d on board %q: %w", sprintID, boardName, err)
	}
	return reduceSprintReport(sprintID, payload.Contents.Completed, payload.Contents.NotCompleted, payload.Contents.Punted, payload.Contents.CompletedOther, payload.Contents.Added), nil
}

func reduceSprintReport(sprintID int, completed, notCompleted, punted, completedOther []reportIssue, added map[string]bool) *SprintScope {
	out := &SprintScope{SprintID: sprintID, IssuesDone: len(completed), IssuesRemoved: len(punted)}
	for _, group := range []struct {
		issues []reportIssue
		kept   bool // still in the sprint
	}{{completed, true}, {notCompleted, true}, {completedOther, true}, {punted, false}} {
		for _, is := range group.issues {
			wasAdded := added[is.Key]
			if !wasAdded {
				out.Committed += is.initial()
			}
			if !group.kept {
				if !wasAdded {
					out.Removed += is.initial()
				}
				continue
			}
			if wasAdded {
				out.Added += is.initial()
				out.IssuesAdded++
			}
			out.Reestimated += is.current() - is.initial()
			out.Final += is.current()
		}
	}
	for _, is := range completed {
		out.Completed += is.current()
	}
	for _, group := range [][]reportIssue{notCompleted, completedOther} {
		for _, is := range group {
			out.CarriedOver += is.current()
			out.IssuesCarried++
		}
	}
	return out
}
