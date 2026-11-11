package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"go-word-create/internal/config"
	"go-word-create/internal/deliverystats"
	"go-word-create/internal/jiraservice"
)

func main() {
	team := flag.String("team", "", "Team component name (default: all configured teams)")
	sprints := flag.Int("sprints", 0, "Number of closed sprints to report (default: DELIVERY_SPRINT_COUNT)")
	refresh := flag.Bool("refresh", false, "Bypass the in-memory cache (persisted closed-sprint values are still reused)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	jiraSvc, err := jiraservice.NewJiraService(cfg.JiraURL, cfg.JiraUsername, cfg.JiraAPIToken, cfg.JiraEpicField, cfg.JiraSPField)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create Jira service: %v\n", err)
		os.Exit(1)
	}
	svc := deliverystats.New(jiraSvc, deliverystats.OptionsFromConfig(cfg))

	var reports []*deliverystats.Report
	for _, t := range deliverystats.TeamsFromConfig(cfg) {
		if *team != "" && t.Component != *team {
			continue
		}
		rep, err := svc.Report(t, *sprints, *refresh)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: team %q: %v\n", t.Component, err)
			os.Exit(1)
		}
		reports = append(reports, rep)
	}
	if len(reports) == 0 {
		fmt.Fprintf(os.Stderr, "Error: no configured team matches -team %q\n", *team)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(reports); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to write report: %v\n", err)
		os.Exit(1)
	}
}
