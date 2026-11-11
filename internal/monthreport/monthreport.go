// Package monthreport builds the month report (issues "In Progress" during a
// month, per team, Closed vs Open) as a Word document. It is shared by the
// get-month-issues CLI and the MCP month_issues tool.
package monthreport

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go-word-create/internal/jiraservice"
	"go-word-create/internal/word"
)

// ClosedStatus is the exact status that counts an issue as delivered.
const ClosedStatus = "Closed"

// TeamIssues groups the issues of one team (Jira component) for a month.
type TeamIssues struct {
	Component string
	Closed    []jiraservice.Issue
	Open      []jiraservice.Issue
}

// Split divides a team's issues into Closed (status exactly "Closed") and Open (everything else).
func Split(component string, issues []jiraservice.Issue) TeamIssues {
	ti := TeamIssues{Component: component}
	for _, iss := range issues {
		if iss.Status == ClosedStatus {
			ti.Closed = append(ti.Closed, iss)
		} else {
			ti.Open = append(ti.Open, iss)
		}
	}
	return ti
}

// BuildDocument creates the Word document: a Closed and an Open table per team.
func BuildDocument(month time.Time, teams []TeamIssues) *word.Doc {
	doc := word.NewDocument()
	label := month.Format("January 2006")
	for _, t := range teams {
		addTable(doc, fmt.Sprintf("Closed Issues During %s (%s)", label, t.Component), t.Closed)
		addTable(doc, fmt.Sprintf("Issues were in work but not Closed during %s (%s)", label, t.Component), t.Open)
	}
	return doc
}

// Bytes renders the month report to .docx bytes.
func Bytes(month time.Time, teams []TeamIssues) ([]byte, error) {
	var buf bytes.Buffer
	if err := BuildDocument(month, teams).Write(&buf); err != nil {
		return nil, fmt.Errorf("month %s: failed to render document: %w", month.Format("2006.01"), err)
	}
	return buf.Bytes(), nil
}

// FileName inserts "- YYYY-MM" before the .docx extension of base,
// e.g. "EPAM-delivered-scope.docx" -> "EPAM-delivered-scope - 2025-10.docx".
func FileName(base string, month time.Time) string {
	return fmt.Sprintf("%s - %s.docx", strings.TrimSuffix(base, ".docx"), month.Format("2006-01"))
}

func addTable(doc *word.Doc, heading string, issues []jiraservice.Issue) {
	doc.AddHeading(1, heading)

	table := word.NewTable(&doc.WordDocument, word.DefaultConfig())
	table.AddHeaderRow()
	for _, issue := range issues {
		table.AddDataRow([]string{
			issue.Type,
			issue.Key,
			issue.Summary,
			issue.Epic,
			strconv.FormatFloat(issue.StoryPoints, 'f', 1, 64),
		})
	}
}
