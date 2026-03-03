package handlers

import (
	"net/http"
	"strings"
)

type nodeInfoLink struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

type nodeInfoWellKnownResponse struct {
	Links []nodeInfoLink `json:"links"`
}

type nodeInfoSoftware struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type nodeInfoServices struct {
	Inbound  []string `json:"inbound"`
	Outbound []string `json:"outbound"`
}

type nodeInfoUsers struct {
	Total          int64 `json:"total"`
	ActiveMonth    int64 `json:"activeMonth"`
	ActiveHalfyear int64 `json:"activeHalfyear"`
}

type nodeInfoUsage struct {
	Users        nodeInfoUsers `json:"users"`
	LocalPosts   int64         `json:"localPosts"`
	LocalComment int64         `json:"localComments"`
}

type nodeInfoResponse struct {
	Version           string           `json:"version"`
	Software          nodeInfoSoftware `json:"software"`
	Protocols         []string         `json:"protocols"`
	Services          nodeInfoServices `json:"services"`
	OpenRegistrations bool             `json:"openRegistrations"`
	Usage             nodeInfoUsage    `json:"usage"`
	Metadata          map[string]any   `json:"metadata"`
}

func NodeInfoWellKnown(deps Dependencies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		base := strings.TrimRight(deps.Config.AppBaseURL, "/")
		writeJSON(w, http.StatusOK, nodeInfoWellKnownResponse{
			Links: []nodeInfoLink{
				{
					Rel:  "http://nodeinfo.diaspora.software/ns/schema/2.0",
					Href: base + "/nodeinfo/2.0",
				},
			},
		})
	})
}

func NodeInfo20(deps Dependencies) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var totalUsers int64
		var localPosts int64

		if err := deps.PG.QueryRow(r.Context(), `SELECT COUNT(1) FROM actors WHERE local = TRUE`).Scan(&totalUsers); err != nil {
			deps.Logger.Error("nodeinfo users count failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "internal_server_error",
			})
			return
		}

		if err := deps.PG.QueryRow(r.Context(), `SELECT COUNT(1) FROM notes WHERE local = TRUE`).Scan(&localPosts); err != nil {
			deps.Logger.Error("nodeinfo post count failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "internal_server_error",
			})
			return
		}

		writeJSON(w, http.StatusOK, nodeInfoResponse{
			Version: "2.0",
			Software: nodeInfoSoftware{
				Name:    "gnusocial-modern",
				Version: "0.1.0",
			},
			Protocols: []string{"activitypub"},
			Services: nodeInfoServices{
				Inbound:  []string{},
				Outbound: []string{},
			},
			OpenRegistrations: false,
			Usage: nodeInfoUsage{
				Users: nodeInfoUsers{
					Total:          totalUsers,
					ActiveMonth:    totalUsers,
					ActiveHalfyear: totalUsers,
				},
				LocalPosts:   localPosts,
				LocalComment: 0,
			},
			Metadata: map[string]any{},
		})
	})
}
