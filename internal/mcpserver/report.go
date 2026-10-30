package mcpserver

import (
	"math"
	"sort"
	"strings"

	"go-word-create/internal/jiraservice"
	"go-word-create/internal/labelreport"
	"go-word-create/internal/sprintview"
)

// LabelBucket is the number of issues carrying one label and their story points.
type LabelBucket struct {
	Label       string  `json:"label"`
	Count       int     `json:"count"`
	StoryPoints float64 `json:"storyPoints"`
}

// Bucket is an issue count with story points.
type Bucket struct {
	Count       int     `json:"count"`
	StoryPoints float64 `json:"storyPoints"`
}

// ReportIssue is the compact issue form used inside label reports.
type ReportIssue struct {
	Key         string  `json:"key"`
	Summary     string  `json:"summary"`
	StoryPoints float64 `json:"storyPoints"`
}

// ReportLabelGroup is one configured label inside a component report.
type ReportLabelGroup struct {
	Label          string        `json:"label"`
	Count          int           `json:"count"`
	CountPct       float64       `json:"countPct"`
	StoryPoints    float64       `json:"storyPoints"`
	StoryPointsPct float64       `json:"storyPointsPct"`
	Issues         []ReportIssue `json:"issues"`
}

// ReportComponent is the label report of one Jira component within a sprint.
type ReportComponent struct {
	Component             string             `json:"component"`
	IssueCount            int                `json:"issueCount"`
	StoryPoints           float64            `json:"storyPoints"`
	LabelGroups           []ReportLabelGroup `json:"labelGroups"`
	UnlabeledIssues       []ReportIssue      `json:"unlabeledIssues"`
	LabeledCountPct       float64            `json:"labeledCountPct"`
	LabeledStoryPointsPct float64            `json:"labeledStoryPointsPct"`
}

// SprintLabelReport is the label report of one sprint.
type SprintLabelReport struct {
	Sprint     string            `json:"sprint"`
	Components []ReportComponent `json:"components"`
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

// filterByStatus keeps issues whose status matches any of statuses (case-insensitive,
// trimmed). It also returns the requested statuses that matched no issue.
func filterByStatus[T any](issues []T, statuses []string, statusOf func(T) string) (kept []T, unmatched []string) {
	if len(statuses) == 0 {
		return issues, nil
	}
	wanted := make(map[string]string, len(statuses))
	for _, s := range statuses {
		if t := strings.TrimSpace(s); t != "" {
			wanted[strings.ToLower(t)] = t
		}
	}
	if len(wanted) == 0 {
		return issues, nil
	}
	seen := make(map[string]bool, len(wanted))
	kept = []T{}
	for _, iss := range issues {
		k := strings.ToLower(strings.TrimSpace(statusOf(iss)))
		if _, ok := wanted[k]; ok {
			kept = append(kept, iss)
			seen[k] = true
		}
	}
	for k, orig := range wanted {
		if !seen[k] {
			unmatched = append(unmatched, orig)
		}
	}
	sort.Strings(unmatched)
	return kept, unmatched
}

// availableStatuses returns the distinct issue statuses, sorted.
func availableStatuses(issues []sprintview.Issue) []string {
	return distinctSorted(issues, func(i sprintview.Issue) string { return i.Status })
}

func statusesOf(issues []jiraservice.Issue) []string {
	return distinctSorted(issues, func(i jiraservice.Issue) string { return i.Status })
}

func distinctSorted[T any](items []T, key func(T) string) []string {
	set := map[string]struct{}{}
	for _, it := range items {
		set[key(it)] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// summarize counts issues and story points per configured label (non-exclusive),
// plus overall totals and the issues carrying none of the configured labels.
func summarize(issues []sprintview.Issue, labels []string) (perLabel []LabelBucket, total, unlabeled Bucket) {
	idx := make(map[string]int, len(labels))
	perLabel = make([]LabelBucket, len(labels))
	for i, l := range labels {
		idx[l] = i
		perLabel[i].Label = l
	}
	for _, iss := range issues {
		total.Count++
		total.StoryPoints += iss.StoryPoints
		if len(iss.ActiveLabels) == 0 {
			unlabeled.Count++
			unlabeled.StoryPoints += iss.StoryPoints
		}
		for _, l := range iss.ActiveLabels {
			if i, ok := idx[l]; ok {
				perLabel[i].Count++
				perLabel[i].StoryPoints += iss.StoryPoints
			}
		}
	}
	return perLabel, total, unlabeled
}

func toReportIssues(issues []jiraservice.Issue) []ReportIssue {
	out := make([]ReportIssue, 0, len(issues))
	for _, iss := range issues {
		out = append(out, ReportIssue{Key: iss.Key, Summary: iss.Summary, StoryPoints: iss.StoryPoints})
	}
	return out
}

// buildLabelReport aggregates sprint issues into per-component label reports.
// A non-empty component restricts the result to that component.
func buildLabelReport(sprint string, issues []jiraservice.Issue, labels []string, component string) SprintLabelReport {
	out := SprintLabelReport{Sprint: sprint, Components: []ReportComponent{}}
	for _, r := range labelreport.Aggregate(issues, labels) {
		if component != "" && r.ComponentName != component {
			continue
		}

		uniq := map[string]float64{}
		groups := make([]ReportLabelGroup, 0, len(r.LabelGroups))
		for _, g := range r.LabelGroups {
			for _, iss := range g.Issues {
				uniq[iss.Key] = iss.StoryPoints
			}
			groups = append(groups, ReportLabelGroup{
				Label:          g.LabelName,
				Count:          g.Count,
				CountPct:       round1(g.CountPct),
				StoryPoints:    g.TotalSP,
				StoryPointsPct: round1(g.TotalSPPct),
				Issues:         toReportIssues(g.Issues),
			})
		}
		for _, iss := range r.UnlabeledIssues {
			uniq[iss.Key] = iss.StoryPoints
		}
		totalSP := 0.0
		for _, sp := range uniq {
			totalSP += sp
		}

		out.Components = append(out.Components, ReportComponent{
			Component:             r.ComponentName,
			IssueCount:            len(uniq),
			StoryPoints:           totalSP,
			LabelGroups:           groups,
			UnlabeledIssues:       toReportIssues(r.UnlabeledIssues),
			LabeledCountPct:       round1(r.TotalLabeledCountPct),
			LabeledStoryPointsPct: round1(r.TotalLabeledSPPct),
		})
	}
	return out
}
