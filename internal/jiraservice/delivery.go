package jiraservice

import (
	"fmt"
	"sort"
	"strings"
	"time"

	jira "github.com/andygrunwald/go-jira"
)

// SprintInfo is a closed or active sprint with the dates needed for delivery stats.
type SprintInfo struct {
	ID       int
	Name     string
	State    string
	Start    time.Time
	End      time.Time // planned end
	Complete time.Time // actual close time; zero while the sprint is active
}

// SprintVelocity is one row of the Jira board velocity chart: story points
// committed when the sprint started versus completed by its end.
type SprintVelocity struct {
	SprintID  int
	Name      string
	Committed float64
	Completed float64
}

// Change is one changelog item of an issue (a single field transition).
type Change struct {
	At         time.Time
	Field      string
	From       string
	To         string
	FromString string
	ToString   string
}

// HistoryIssue is the slim, time-aware view of an issue used by delivery stats.
type HistoryIssue struct {
	Key            string
	Type           string
	Status         string
	StatusCategory string // "done", "new" or "indeterminate"; empty when unknown
	Priority       string
	Resolution     string
	StoryPoints    float64
	Created        time.Time
	Resolved       time.Time // Jira's resolution date; zero while unresolved (see deliverystats for closed-without-resolution issues)
	Changes        []Change  // ascending by time; empty unless the changelog was requested
}

// ClosedSprints returns the closed sprints of the named board in chronological
// order (by start date; sprint IDs are not reliably chronological). A positive
// limit keeps only the newest `limit` ones.
func (s *JiraService) ClosedSprints(boardName string, limit int) ([]SprintInfo, error) {
	all, err := s.GetSprintsForBoard(boardName, 0)
	if err != nil {
		return nil, err
	}
	var closed []SprintInfo
	for _, sp := range all {
		if !strings.EqualFold(sp.State, "closed") || sp.StartDate == nil {
			continue
		}
		info := SprintInfo{ID: sp.ID, Name: sp.Name, State: sp.State, Start: *sp.StartDate}
		if sp.EndDate != nil {
			info.End = *sp.EndDate
		}
		if sp.CompleteDate != nil {
			info.Complete = *sp.CompleteDate
		}
		closed = append(closed, info)
	}
	sort.SliceStable(closed, func(i, j int) bool {
		if !closed[i].Start.Equal(closed[j].Start) {
			return closed[i].Start.Before(closed[j].Start)
		}
		return closed[i].ID < closed[j].ID
	})
	if limit > 0 && len(closed) > limit {
		closed = closed[len(closed)-limit:]
	}
	return closed, nil
}

// BoardVelocity reads the Jira velocity chart of the named board and returns
// its rows keyed by sprint ID. It uses the Jira Software "greenhopper" chart
// endpoint, which is not part of the documented REST API.
func (s *JiraService) BoardVelocity(boardName string) (map[int]SprintVelocity, error) {
	board, err := s.GetBoard(boardName)
	if err != nil {
		return nil, err
	}
	req, err := s.client.NewRequest("GET", fmt.Sprintf("rest/greenhopper/1.0/rapid/charts/velocity?rapidViewId=%d", board.ID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build velocity request for board %q: %w", boardName, err)
	}
	var payload struct {
		Sprints []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"sprints"`
		Entries map[string]struct {
			Estimated struct {
				Value float64 `json:"value"`
			} `json:"estimated"`
			Completed struct {
				Value float64 `json:"value"`
			} `json:"completed"`
		} `json:"velocityStatEntries"`
	}
	if _, err := s.client.Do(req, &payload); err != nil {
		return nil, fmt.Errorf("failed to read velocity chart of board %q: %w", boardName, err)
	}
	out := make(map[int]SprintVelocity, len(payload.Sprints))
	for _, sp := range payload.Sprints {
		e := payload.Entries[fmt.Sprint(sp.ID)]
		out[sp.ID] = SprintVelocity{SprintID: sp.ID, Name: sp.Name, Committed: e.Estimated.Value, Completed: e.Completed.Value}
	}
	return out, nil
}

// SearchHistoryIssues runs the JQL and returns every match (all pages) as
// HistoryIssue. With withChangelog the field transitions are included.
func (s *JiraService) SearchHistoryIssues(jql string, withChangelog bool) ([]HistoryIssue, error) {
	opts := &jira.SearchOptions{
		MaxResults: 100,
		Fields:     []string{"issuetype", "status", "priority", "resolution", "resolutiondate", "created", s.spField},
	}
	if withChangelog {
		opts.Expand = "changelog"
	}
	var out []HistoryIssue
	err := s.client.Issue.SearchPages(jql, opts, func(is jira.Issue) error {
		out = append(out, s.toHistoryIssue(is))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search issues (jql=%q): %w", jql, err)
	}
	return out, nil
}

func (s *JiraService) toHistoryIssue(is jira.Issue) HistoryIssue {
	h := HistoryIssue{
		Key:         is.Key,
		Type:        is.Fields.Type.Name,
		StoryPoints: getStoryPoints(is, s.spField),
		Created:     time.Time(is.Fields.Created),
		Resolved:    time.Time(is.Fields.Resolutiondate),
	}
	if is.Fields.Status != nil {
		h.Status = is.Fields.Status.Name
		h.StatusCategory = strings.ToLower(is.Fields.Status.StatusCategory.Key)
	}
	if is.Fields.Priority != nil {
		h.Priority = is.Fields.Priority.Name
	}
	if is.Fields.Resolution != nil {
		h.Resolution = is.Fields.Resolution.Name
	}
	if is.Changelog != nil {
		for _, hist := range is.Changelog.Histories {
			at, ok := parseJiraTime(hist.Created)
			if !ok {
				continue
			}
			for _, it := range hist.Items {
				h.Changes = append(h.Changes, Change{
					At:         at,
					Field:      it.Field,
					From:       fmt.Sprint(nilToEmpty(it.From)),
					To:         fmt.Sprint(nilToEmpty(it.To)),
					FromString: it.FromString,
					ToString:   it.ToString,
				})
			}
		}
		sort.SliceStable(h.Changes, func(i, j int) bool { return h.Changes[i].At.Before(h.Changes[j].At) })
	}
	return h
}

func nilToEmpty(v interface{}) interface{} {
	if v == nil {
		return ""
	}
	return v
}

func parseJiraTime(s string) (time.Time, bool) {
	for _, layout := range []string{"2006-01-02T15:04:05.000-0700", time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
