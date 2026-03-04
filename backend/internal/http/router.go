package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	aphandlers "github.com/cryptic-stack/special-goggles/backend/internal/ap/activitypub/handlers"
	"github.com/cryptic-stack/special-goggles/backend/internal/config"
	"github.com/cryptic-stack/special-goggles/backend/internal/http/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	Config config.Config
	Logger *slog.Logger
	PG     *pgxpool.Pool
}

func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()

	apDeps := aphandlers.Dependencies{
		Config: deps.Config,
		Logger: deps.Logger,
		PG:     deps.PG,
	}

	mux.Handle("GET /.well-known/webfinger", aphandlers.WebFinger(apDeps))
	mux.Handle("GET /.well-known/nodeinfo", aphandlers.NodeInfoWellKnown(apDeps))
	mux.Handle("GET /nodeinfo/2.0", aphandlers.NodeInfo20(apDeps))
	mux.Handle("GET /users/{username}", aphandlers.Actor(apDeps))
	mux.Handle("GET /users/{username}/outbox", aphandlers.Outbox(apDeps))
	mux.Handle("GET /users/{username}/followers", aphandlers.Followers(apDeps))
	mux.Handle("GET /users/{username}/following", aphandlers.Following(apDeps))
	mux.Handle("POST /users/{username}/inbox", aphandlers.Inbox(apDeps))
	mux.Handle("GET /notes/{id}", aphandlers.NoteObject(apDeps))

	mux.Handle("POST /auth/register", handleAuthRegister(deps))
	mux.Handle("POST /auth/login", handleAuthLogin(deps))
	mux.Handle("POST /auth/logout", handleAuthLogout(deps))
	mux.Handle("GET /auth/me", handleAuthMe(deps))

	mux.Handle("POST /api/v1/posts", handleCreatePost(deps))
	mux.Handle("POST /api/v1/media", handleUploadMedia(deps))
	mux.Handle("GET /api/v1/settings/theme", handleGetThemeSettings(deps))
	mux.Handle("PUT /api/v1/settings/theme", handlePutThemeSettings(deps))
	mux.Handle("DELETE /api/v1/posts/{id}", handleDeleteOwnPost(deps))
	mux.Handle("GET /api/v1/timelines/home", handleHomeTimeline(deps))
	mux.Handle("GET /api/v1/timelines/local", handleLocalTimeline(deps))
	mux.Handle("POST /api/v1/follows", handleFollow(deps))
	mux.Handle("POST /api/v1/unfollow", handleUnfollow(deps))
	mux.Handle("POST /api/v1/notes/{id}/like", handleCreateReaction(deps, "like", "Like"))
	mux.Handle("DELETE /api/v1/notes/{id}/like", handleDeleteReaction(deps, "like"))
	mux.Handle("POST /api/v1/notes/{id}/boost", handleCreateReaction(deps, "announce", "Announce"))
	mux.Handle("DELETE /api/v1/notes/{id}/boost", handleDeleteReaction(deps, "announce"))
	mux.Handle("GET /api/v1/notifications", handleListNotifications(deps))
	mux.Handle("POST /api/v1/notifications/read-all", handleReadAllNotifications(deps))
	mux.Handle("POST /api/v1/groups", handleCreateGroup(deps))
	mux.Handle("POST /api/v1/groups/{slug}/join", handleJoinGroup(deps))
	mux.Handle("POST /api/v1/groups/{slug}/posts", handleCreateGroupPost(deps))
	mux.Handle("GET /api/v1/groups/{slug}/timeline", handleGroupTimeline(deps))
	mux.Handle("POST /api/v1/reports", handleCreateReport(deps))
	mux.Handle("GET /api/v1/admin/reports", requireAdmin(deps, handleListReports(deps)))
	mux.Handle("PUT /api/v1/admin/domain-policies", requireAdmin(deps, handleSetDomainPolicy(deps)))
	mux.Handle("GET /media/", http.StripPrefix("/media/", http.FileServer(http.Dir(filepath.Join(deps.Config.DataDir, "media")))))
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web"))))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		const indexPath = "web/index.html"
		if _, err := os.Stat(indexPath); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				deps.Logger.Error("ui index lookup failed", "error", err)
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"service": "gnusocial-modern",
				"status":  "ok",
				"ui":      "not_built",
			})
			return
		}
		http.ServeFile(w, r, indexPath)
	})

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		requestID := middleware.FromContext(r.Context())
		writeJSON(w, http.StatusOK, map[string]string{
			"status":     "ok",
			"request_id": requestID,
		})
	})

	stack := middleware.Chain(
		middleware.RecoverJSON(deps.Logger),
		middleware.RequestID(),
		middleware.AccessLog(deps.Logger),
		middleware.EnforcePublicHost(deps.Config.AppEnv, deps.Config.AppDomain),
		middleware.RateLimit(deps.Config.RateLimitRPS, deps.Config.RateLimitBurst),
	)

	return stack(mux)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
