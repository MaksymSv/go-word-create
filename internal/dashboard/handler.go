package dashboard

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"time"

	"go-word-create/internal/config"
	"go-word-create/internal/jiraservice"
)

//go:embed web
var webFiles embed.FS

// DashboardHandler serves the sprint labels web dashboard.
type DashboardHandler struct {
	cfg   *config.Config
	jira  *jiraservice.JiraService
	webFS fs.FS
}

// NewHandler creates a DashboardHandler with a sub-FS rooted at "web/".
func NewHandler(cfg *config.Config, jira *jiraservice.JiraService) *DashboardHandler {
	sub, err := fs.Sub(webFiles, "web")
	if err != nil {
		panic(fmt.Sprintf("dashboard: failed to sub-root web embed: %v", err))
	}
	return &DashboardHandler{cfg: cfg, jira: jira, webFS: sub}
}

// Routes returns an http.Handler with all dashboard routes registered.
func (h *DashboardHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(h.webFS)))
	mux.HandleFunc("GET /", h.serveIndex)
	mux.HandleFunc("GET /api/teams", h.getTeams)
	mux.HandleFunc("GET /api/teams/{component}/sprints", h.getSprints)
	mux.HandleFunc("GET /api/sprints/{sprintID}/issues", h.getSprintIssues)
	mux.HandleFunc("POST /api/issues/{issueKey}/labels", h.updateLabel)
	return loggingMiddleware(mux)
}

func (h *DashboardHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFileFS(w, r, h.webFS, "index.html")
}

func (h *DashboardHandler) getTeams(w http.ResponseWriter, r *http.Request) {
	resp := make([]TeamResponse, len(h.cfg.Teams))
	for i, t := range h.cfg.Teams {
		resp[i] = TeamResponse{ComponentName: t.ComponentName, BoardName: t.BoardName}
	}
	writeJSON(w, resp)
}

func (h *DashboardHandler) getSprints(w http.ResponseWriter, r *http.Request) {
	component := r.PathValue("component")
	team := h.cfg.GetTeamByComponent(component)
	if team.ComponentName == "" {
		writeError(w, http.StatusNotFound, fmt.Sprintf("team %q not found", component))
		return
	}
	sprints, err := h.jira.GetSprintsForBoard(team.BoardName, 5)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("failed to get sprints for board %q: %v", team.BoardName, err))
		return
	}
	resp := make([]SprintResponse, len(sprints))
	for i, s := range sprints {
		resp[i] = SprintResponse{
			ID:        s.ID,
			Name:      s.Name,
			State:     s.State,
			StartDate: s.StartDate,
			EndDate:   s.EndDate,
		}
	}
	writeJSON(w, resp)
}

func (h *DashboardHandler) getSprintIssues(w http.ResponseWriter, r *http.Request) {
	sprintIDStr := r.PathValue("sprintID")
	sprintID, err := strconv.Atoi(sprintIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid sprint ID")
		return
	}

	epicNames, err := h.jira.LoadEpics(h.cfg.ProjectKey)
	if err != nil {
		log.Printf("dashboard: failed to load epics: %v", err)
		epicNames = make(map[string]string)
	}

	issues, err := h.jira.LoadIssuesFromSprint(sprintID, epicNames, nil)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("failed to load sprint issues (sprintID=%d): %v", sprintID, err))
		return
	}

	configuredSet := make(map[string]struct{}, len(h.cfg.ReportLabels))
	for _, l := range h.cfg.ReportLabels {
		configuredSet[l] = struct{}{}
	}

	dashIssues := make([]DashboardIssue, 0, len(issues))
	for _, issue := range issues {
		active := []string{}
		for _, l := range issue.Labels {
			if _, ok := configuredSet[l]; ok {
				active = append(active, l)
			}
		}
		dashIssues = append(dashIssues, DashboardIssue{
			Key:          issue.Key,
			Summary:      issue.Summary,
			Epic:         issue.Epic,
			StoryPoints:  issue.StoryPoints,
			Type:         issue.Type,
			Status:       issue.Status,
			URL:          issue.URL,
			ActiveLabels: active,
		})
	}

	writeJSON(w, SprintIssuesResponse{
		ConfiguredLabels: h.cfg.ReportLabels,
		Issues:           dashIssues,
	})
}

func (h *DashboardHandler) updateLabel(w http.ResponseWriter, r *http.Request) {
	issueKey := r.PathValue("issueKey")
	var req LabelUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	if req.Action != "add" && req.Action != "remove" {
		writeError(w, http.StatusBadRequest, `action must be "add" or "remove"`)
		return
	}
	if req.Label == "" {
		writeError(w, http.StatusBadRequest, "label is required")
		return
	}
	allowed := false
	for _, l := range h.cfg.ReportLabels {
		if l == req.Label {
			allowed = true
			break
		}
	}
	if !allowed {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("label %q is not in the configured label list", req.Label))
		return
	}
	if err := h.jira.UpdateIssueLabel(issueKey, req.Label, req.Action == "add"); err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("failed to update label: %v", err))
		return
	}
	w.WriteHeader(http.StatusOK)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rec.status, time.Since(start))
	})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("dashboard: failed to encode JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
