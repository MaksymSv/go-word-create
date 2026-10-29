package jiraservice

import (
	"fmt"
	"strings"
)

// ParseSprintNames splits a comma-separated list of sprint names, trimming whitespace.
func ParseSprintNames(raw string) ([]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("sprint flag is empty")
	}

	parts := strings.Split(trimmed, ",")
	sprintNames := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			return nil, fmt.Errorf("sprint list contains an empty sprint name")
		}
		sprintNames = append(sprintNames, name)
	}

	return sprintNames, nil
}
