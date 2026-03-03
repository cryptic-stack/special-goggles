package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv                 string
	AppBaseURL             string
	AppDomain              string
	AppListenAddr          string
	MigrationsDir          string
	InboxMaxBody           int64
	APAllowUnsignedInbound bool
	APSignatureMaxSkewSec  int

	DBDSN     string
	RedisAddr string
	DataDir   string

	SessionSecret string
	JWTIssuer     string
	JWTAudience   string
	JWTSigningKey string

	RateLimitRPS   float64
	RateLimitBurst int

	DevSeedPassword string
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:                 getEnv("APP_ENV", "dev"),
		AppBaseURL:             os.Getenv("APP_BASE_URL"),
		AppDomain:              os.Getenv("APP_DOMAIN"),
		AppListenAddr:          getEnv("APP_LISTEN_ADDR", ":8080"),
		MigrationsDir:          getEnv("MIGRATIONS_DIR", "./migrations"),
		InboxMaxBody:           getEnvInt64("INBOX_MAX_BODY_BYTES", 1<<20),
		APAllowUnsignedInbound: getEnvBool("AP_ALLOW_UNSIGNED_INBOUND", true),
		APSignatureMaxSkewSec:  getEnvInt("AP_SIGNATURE_MAX_SKEW_SECONDS", 300),
		DBDSN:                  os.Getenv("DB_DSN"),
		RedisAddr:              os.Getenv("REDIS_ADDR"),
		DataDir:                getEnv("DATA_DIR", "/data"),
		SessionSecret:          os.Getenv("SESSION_SECRET"),
		JWTIssuer:              os.Getenv("JWT_ISSUER"),
		JWTAudience:            os.Getenv("JWT_AUDIENCE"),
		JWTSigningKey:          os.Getenv("JWT_SIGNING_KEY"),
		RateLimitRPS:           getEnvFloat("RATE_LIMIT_RPS", 10),
		RateLimitBurst:         getEnvInt("RATE_LIMIT_BURST", 30),
		DevSeedPassword:        getEnv("DEV_SEED_PASSWORD", "alice12345"),
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) validate() error {
	required := map[string]string{
		"APP_BASE_URL":    c.AppBaseURL,
		"APP_DOMAIN":      c.AppDomain,
		"MIGRATIONS_DIR":  c.MigrationsDir,
		"DB_DSN":          c.DBDSN,
		"REDIS_ADDR":      c.RedisAddr,
		"SESSION_SECRET":  c.SessionSecret,
		"JWT_ISSUER":      c.JWTIssuer,
		"JWT_AUDIENCE":    c.JWTAudience,
		"JWT_SIGNING_KEY": c.JWTSigningKey,
	}

	var missing []string
	for key, val := range required {
		if val == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missing)
	}

	if c.RateLimitRPS <= 0 {
		return errors.New("RATE_LIMIT_RPS must be > 0")
	}
	if c.RateLimitBurst <= 0 {
		return errors.New("RATE_LIMIT_BURST must be > 0")
	}
	if c.InboxMaxBody <= 0 {
		return errors.New("INBOX_MAX_BODY_BYTES must be > 0")
	}
	if c.APSignatureMaxSkewSec <= 0 {
		return errors.New("AP_SIGNATURE_MAX_SKEW_SECONDS must be > 0")
	}
	if isProdEnv(c.AppEnv) {
		if isLocalhostLikeHost(extractHost(c.AppDomain)) {
			return errors.New("APP_DOMAIN cannot be localhost/loopback in prod")
		}

		baseHost := extractHost(c.AppBaseURL)
		if baseHost == "" {
			return errors.New("APP_BASE_URL must include a host")
		}
		if isLocalhostLikeHost(baseHost) {
			return errors.New("APP_BASE_URL cannot target localhost/loopback in prod")
		}
	}

	return nil
}

func isProdEnv(env string) bool {
	normalized := strings.TrimSpace(strings.ToLower(env))
	return normalized == "prod" || normalized == "production"
}

func extractHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	hostPort := raw
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return ""
		}
		hostPort = parsed.Host
	}

	hostPort = strings.TrimSpace(hostPort)
	if hostPort == "" {
		return ""
	}

	if strings.Contains(hostPort, ":") {
		if h, _, err := net.SplitHostPort(hostPort); err == nil {
			return strings.Trim(strings.ToLower(h), "[]")
		}
	}

	return strings.Trim(strings.ToLower(hostPort), "[]")
}

func isLocalhostLikeHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return false
	}

	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}

	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	parsed, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvInt64(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}

	switch v {
	case "1", "true", "t", "yes", "y", "on":
		return true
	case "0", "false", "f", "no", "n", "off":
		return false
	default:
		return fallback
	}
}
