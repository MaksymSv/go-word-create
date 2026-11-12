package dashboard

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"go-word-create/internal/raidscan"
)

// RaidSettings tells the RAID page which options it may offer.
type RaidSettings struct {
	DriftPct int    `json:"driftPct"`
	JiraURL  string `json:"jiraUrl"`
}

func (h *DashboardHandler) serveRaid(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, h.webFS, "raid.html")
}

func (h *DashboardHandler) getRaidSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, RaidSettings{
		DriftPct: h.raid.DriftPct(),
		JiraURL:  strings.TrimRight(h.cfg.JiraURL, "/"),
	})
}

func (h *DashboardHandler) getRaidReport(w http.ResponseWriter, r *http.Request) {
	component := r.PathValue("component")
	team := h.cfg.GetTeamByComponent(component)
	if team.ComponentName == "" {
		writeError(w, http.StatusNotFound, fmt.Sprintf("team %q not found", component))
		return
	}
	sprints := 0
	if v := r.URL.Query().Get("sprints"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > raidscan.MaxSprints {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("sprints must be an integer between 1 and %d", raidscan.MaxSprints))
			return
		}
		sprints = n
	}
	writeJSON(w, h.raid.Report(raidscan.Team{Component: team.ComponentName, Board: team.BoardName}, sprints, r.URL.Query().Get("refresh") == "1"))
}

// WarmRaid builds the default RAID report of every team in the background so
// the first page view does not wait for the Jira scan.
func (h *DashboardHandler) WarmRaid() {
	go func() {
		for _, t := range raidscan.TeamsFromConfig(h.cfg) {
			h.raid.Report(t, 0, false)
		}
		log.Printf("dashboard: RAID reports warmed")
	}()
}
