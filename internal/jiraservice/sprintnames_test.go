package jiraservice

import "testing"

func TestParseSprintNames(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    []string
		wantErr bool
	}{
		{name: "single sprint", raw: "Sprint 16", want: []string{"Sprint 16"}},
		{name: "comma separated sprints", raw: "PC26.13,ST26.13", want: []string{"PC26.13", "ST26.13"}},
		{name: "trim whitespace", raw: "  Sprint 16 , ST26.13  ", want: []string{"Sprint 16", "ST26.13"}},
		{name: "empty sprint", raw: "Sprint 16,", wantErr: true},
		{name: "empty input", raw: "   ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSprintNames(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseSprintNames(%q) expected error, got nil", tt.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseSprintNames(%q) unexpected error: %v", tt.raw, err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ParseSprintNames(%q) returned %d names, want %d: %v", tt.raw, len(got), len(tt.want), got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("ParseSprintNames(%q) got %q at index %d, want %q", tt.raw, got[i], i, tt.want[i])
				}
			}
		})
	}
}
