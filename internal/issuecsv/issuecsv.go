// Package issuecsv renders sprint issues as CSV; shared by the CLI and the MCP server.
package issuecsv

import (
	"encoding/csv"
	"io"
	"regexp"
	"strconv"
	"time"

	"go-word-create/internal/jiraservice"
)

var unsafeFileChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// FileName returns "<sprint>_<YYYY-MM-DD>.csv" with unsafe characters in the sprint name replaced by "_".
func FileName(sprint string, generatedAt time.Time) string {
	return unsafeFileChars.ReplaceAllString(sprint, "_") + "_" + generatedAt.Format("2006-01-02") + ".csv"
}

// Write writes a header row and one row per issue: Type, Key, Summary, Epic, Story Points, Status.
func Write(w io.Writer, issues []jiraservice.Issue) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"Type", "Key", "Summary", "Epic", "Story Points", "Status"}); err != nil {
		return err
	}
	for _, issue := range issues {
		row := []string{
			issue.Type,
			issue.Key,
			issue.Summary,
			issue.Epic,
			strconv.FormatFloat(issue.StoryPoints, 'f', 1, 64),
			issue.Status,
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
