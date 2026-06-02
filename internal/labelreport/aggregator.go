package labelreport

import (
	"sort"

	"go-word-create/internal/jiraservice"
)

type LabelGroup struct {
	LabelName  string
	Issues     []jiraservice.Issue
	Count      int
	TotalSP    float64
	CountPct   float64
	TotalSPPct float64
}

type ComponentReport struct {
	ComponentName        string
	LabelGroups          []LabelGroup
	UnlabeledIssues      []jiraservice.Issue
	TotalLabeledCountPct float64
	TotalLabeledSPPct    float64
}

// Aggregate groups issues into per-component label reports.
// orderedLabels defines which labels to track and their display order.
// Each issue may appear in multiple LabelGroups (non-exclusive membership).
// Issues with no component are placed under "No Component" (sorted last).
func Aggregate(issues []jiraservice.Issue, orderedLabels []string) []ComponentReport {
	// Map component name → issues belonging to that component
	compIssues := make(map[string][]jiraservice.Issue)
	for _, issue := range issues {
		if len(issue.Components) == 0 {
			compIssues["No Component"] = append(compIssues["No Component"], issue)
		} else {
			for _, comp := range issue.Components {
				compIssues[comp] = append(compIssues[comp], issue)
			}
		}
	}

	// Sort component names alphabetically; "No Component" goes last
	var compNames []string
	for name := range compIssues {
		if name != "No Component" {
			compNames = append(compNames, name)
		}
	}
	sort.Strings(compNames)
	if _, ok := compIssues["No Component"]; ok {
		compNames = append(compNames, "No Component")
	}

	// Build a quick index: label string → position in orderedLabels
	labelIdx := make(map[string]int, len(orderedLabels))
	for i, l := range orderedLabels {
		labelIdx[l] = i
	}

	var reports []ComponentReport
	for _, compName := range compNames {
		// Initialise one LabelGroup per configured label (in order)
		groups := make([]LabelGroup, len(orderedLabels))
		for i, l := range orderedLabels {
			groups[i] = LabelGroup{LabelName: l}
		}

		// Pre-compute component totals for percentage denominators
		componentTotalCount := len(compIssues[compName])
		componentTotalSP := 0.0
		for _, iss := range compIssues[compName] {
			componentTotalSP += iss.StoryPoints
		}

		var unlabeled []jiraservice.Issue
		for _, issue := range compIssues[compName] {
			matched := false
			for _, issueLabel := range issue.Labels {
				if idx, ok := labelIdx[issueLabel]; ok {
					groups[idx].Issues = append(groups[idx].Issues, issue)
					matched = true
				}
			}
			if !matched {
				unlabeled = append(unlabeled, issue)
			}
		}

		// Compute Count, TotalSP, and percentages for each group
		for i := range groups {
			groups[i].Count = len(groups[i].Issues)
			total := 0.0
			for _, iss := range groups[i].Issues {
				total += iss.StoryPoints
			}
			groups[i].TotalSP = total
			if componentTotalCount > 0 {
				groups[i].CountPct = float64(groups[i].Count) / float64(componentTotalCount) * 100
			}
			if componentTotalSP > 0 {
				groups[i].TotalSPPct = groups[i].TotalSP / componentTotalSP * 100
			}
		}

		// Compute labeled-union percentages for the "Total" summary row.
		// labeledCount = issues with at least one configured label (total minus unlabeled).
		labeledCount := componentTotalCount - len(unlabeled)
		labeledSP := componentTotalSP
		for _, iss := range unlabeled {
			labeledSP -= iss.StoryPoints
		}
		var totalLabeledCountPct, totalLabeledSPPct float64
		if componentTotalCount > 0 {
			totalLabeledCountPct = float64(labeledCount) / float64(componentTotalCount) * 100
		}
		if componentTotalSP > 0 {
			totalLabeledSPPct = labeledSP / componentTotalSP * 100
		}

		reports = append(reports, ComponentReport{
			ComponentName:        compName,
			LabelGroups:          groups,
			UnlabeledIssues:      unlabeled,
			TotalLabeledCountPct: totalLabeledCountPct,
			TotalLabeledSPPct:    totalLabeledSPPct,
		})
	}

	return reports
}
