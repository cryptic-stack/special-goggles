package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
)

var (
	hexThemeColorPattern = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)
	allowedThemeKeys     = map[string]struct{}{
		"bg":          {},
		"paper":       {},
		"paper-soft":  {},
		"line":        {},
		"line-strong": {},
		"text":        {},
		"muted":       {},
		"brand":       {},
		"brand-deep":  {},
		"brand-cool":  {},
		"warn":        {},
		"ok":          {},
	}
	allowedThemePresets = map[string]struct{}{
		"forest":   {},
		"midnight": {},
		"sunset":   {},
		"custom":   {},
	}
	allowedThemeOptionValues = map[string]map[string]struct{}{
		"font": {
			"modern": {},
			"serif":  {},
			"mono":   {},
		},
		"density": {
			"comfortable": {},
			"compact":     {},
		},
		"corner": {
			"soft":  {},
			"sharp": {},
		},
	}
)

type themeSettingsRequest struct {
	Preset    string            `json:"preset"`
	Variables map[string]string `json:"variables"`
	Options   map[string]string `json:"options"`
}

type themeSettingsResponse struct {
	Preset    string            `json:"preset"`
	Variables map[string]string `json:"variables"`
	Options   map[string]string `json:"options"`
}

func handleGetThemeSettings(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, err := loadSessionPrincipal(r.Context(), deps, r)
		if errors.Is(err, errUnauthorized) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if err != nil {
			deps.Logger.Error("theme settings session lookup failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		response, err := loadThemeSettings(r.Context(), deps, principal.UserID)
		if err != nil {
			deps.Logger.Error("theme settings load failed", "error", err, "user_id", principal.UserID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		writeJSON(w, http.StatusOK, response)
	}
}

func handlePutThemeSettings(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, err := loadSessionPrincipal(r.Context(), deps, r)
		if errors.Is(err, errUnauthorized) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if err != nil {
			deps.Logger.Error("theme settings session lookup failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		payload, err := decodeAuthBody[themeSettingsRequest](w, r)
		if err != nil {
			status := http.StatusBadRequest
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				status = http.StatusRequestEntityTooLarge
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}

		preset, err := normalizeThemePreset(payload.Preset)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		variables, err := sanitizeThemeVariables(payload.Variables)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		options, err := sanitizeThemeOptions(payload.Options)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		variablesJSON, err := json.Marshal(variables)
		if err != nil {
			deps.Logger.Error("theme settings marshal failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}
		optionsJSON, err := json.Marshal(options)
		if err != nil {
			deps.Logger.Error("theme options marshal failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if _, err := deps.PG.Exec(r.Context(), `
INSERT INTO user_theme_settings (user_id, preset, variables_json, options_json, updated_at)
VALUES ($1, $2, $3, $4, now())
ON CONFLICT (user_id)
DO UPDATE SET
  preset = EXCLUDED.preset,
  variables_json = EXCLUDED.variables_json,
  options_json = EXCLUDED.options_json,
  updated_at = now()
`,
			principal.UserID,
			preset,
			variablesJSON,
			optionsJSON,
		); err != nil {
			deps.Logger.Error("theme settings upsert failed", "error", err, "user_id", principal.UserID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		writeJSON(w, http.StatusOK, themeSettingsResponse{
			Preset:    preset,
			Variables: variables,
			Options:   options,
		})
	}
}

func loadThemeSettings(ctx context.Context, deps Dependencies, userID int64) (themeSettingsResponse, error) {
	var (
		preset       string
		variablesRaw []byte
		optionsRaw   []byte
	)
	err := deps.PG.QueryRow(ctx, `
SELECT preset, variables_json::text, COALESCE(options_json, '{}'::jsonb)::text
FROM user_theme_settings
WHERE user_id = $1
LIMIT 1
`,
		userID,
	).Scan(&preset, &variablesRaw, &optionsRaw)
	if errors.Is(err, pgx.ErrNoRows) {
		return themeSettingsResponse{
			Preset:    "forest",
			Variables: map[string]string{},
			Options:   defaultThemeOptions(),
		}, nil
	}
	if err != nil {
		return themeSettingsResponse{}, err
	}

	preset, err = normalizeThemePreset(preset)
	if err != nil {
		preset = "forest"
	}

	var variables map[string]string
	if len(variablesRaw) > 0 {
		_ = json.Unmarshal(variablesRaw, &variables)
	}
	variables, _ = sanitizeThemeVariables(variables)
	var options map[string]string
	if len(optionsRaw) > 0 {
		_ = json.Unmarshal(optionsRaw, &options)
	}
	sanitizedOptions, err := sanitizeThemeOptions(options)
	if err != nil {
		sanitizedOptions = defaultThemeOptions()
	}

	return themeSettingsResponse{
		Preset:    preset,
		Variables: variables,
		Options:   sanitizedOptions,
	}, nil
}

func normalizeThemePreset(raw string) (string, error) {
	preset := strings.ToLower(strings.TrimSpace(raw))
	if preset == "" {
		preset = "forest"
	}
	if _, ok := allowedThemePresets[preset]; !ok {
		return "", errors.New("invalid_theme_preset")
	}
	return preset, nil
}

func sanitizeThemeVariables(input map[string]string) (map[string]string, error) {
	if len(input) == 0 {
		return map[string]string{}, nil
	}
	if len(input) > len(allowedThemeKeys) {
		return nil, errors.New("too_many_theme_variables")
	}

	out := make(map[string]string, len(input))
	for key, value := range input {
		key = strings.ToLower(strings.TrimSpace(key))
		if key == "" {
			continue
		}
		if _, ok := allowedThemeKeys[key]; !ok {
			return nil, errors.New("invalid_theme_variable_key")
		}

		value = strings.TrimSpace(value)
		if !hexThemeColorPattern.MatchString(value) {
			return nil, errors.New("invalid_theme_color")
		}
		out[key] = strings.ToLower(value)
	}
	return out, nil
}

func sanitizeThemeOptions(input map[string]string) (map[string]string, error) {
	if len(input) == 0 {
		return defaultThemeOptions(), nil
	}
	if len(input) > len(allowedThemeOptionValues) {
		return nil, errors.New("too_many_theme_options")
	}

	out := defaultThemeOptions()
	for key, value := range input {
		key = strings.ToLower(strings.TrimSpace(key))
		allowed, ok := allowedThemeOptionValues[key]
		if !ok {
			return nil, errors.New("invalid_theme_option_key")
		}
		value = strings.ToLower(strings.TrimSpace(value))
		if _, ok := allowed[value]; !ok {
			return nil, errors.New("invalid_theme_option_value")
		}
		out[key] = value
	}
	return out, nil
}

func defaultThemeOptions() map[string]string {
	return map[string]string{
		"font":    "modern",
		"density": "comfortable",
		"corner":  "soft",
	}
}
