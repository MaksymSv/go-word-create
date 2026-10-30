package issuecsv

import (
	"strings"
	"testing"
	"time"

	"go-word-create/internal/jiraservice"
)

func TestFileName(t *testing.T) {
	day := time.Date(2026, 10, 30, 15, 4, 5, 0, time.UTC)
	tests := []struct{ sprint, want string }{
		{"ST26.20", "ST26.20_2026-10-30.csv"},
		{"Sprint 16", "Sprint_16_2026-10-30.csv"},
		{"A/B: c", "A_B_c_2026-10-30.csv"},
	}
	for _, tt := range tests {
		if got := FileName(tt.sprint, day); got != tt.want {
			t.Errorf("FileName(%q) = %q, want %q", tt.sprint, got, tt.want)
		}
	}
}

func TestWrite(t *testing.T) {
	issues := []jiraservice.Issue{
		{Type: "Story", Key: "ABC-1", Summary: `Has, comma and "quotes"`, Epic: "E", StoryPoints: 3, Status: "Done"},
	}
	var sb strings.Builder
	if err := Write(&sb, issues); err != nil {
		t.Fatal(err)
	}
	want := "Type,Key,Summary,Epic,Story Points,Status\n" +
		"Story,ABC-1,\"Has, comma and \"\"quotes\"\"\",E,3.0,Done\n"
	if sb.String() != want {
		t.Fatalf("got:\n%s\nwant:\n%s", sb.String(), want)
	}
}
