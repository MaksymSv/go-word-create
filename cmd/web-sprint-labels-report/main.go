package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"go-word-create/internal/config"
	"go-word-create/internal/dashboard"
	"go-word-create/internal/jiraservice"
)

func main() {
	port := flag.Int("port", 8080, "HTTP listen port for the dashboard")
	debug := flag.Bool("debug", false, "enable verbose logging")
	flag.Parse()

	if *debug {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	jiraSvc, err := jiraservice.NewJiraService(
		cfg.JiraURL,
		cfg.JiraUsername,
		cfg.JiraAPIToken,
		cfg.JiraEpicField,
		cfg.JiraSPField,
	)
	if err != nil {
		log.Fatalf("failed to create Jira client: %v", err)
	}

	handler := dashboard.NewHandler(cfg, jiraSvc)
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting dashboard on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, handler.Routes()))
}
