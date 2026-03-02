package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv                   string
	Port                     string
	AllowedOrigin            string
	StoreProvider            string
	SQLitePath               string
	SessionHMACKey           string
	TokenEncryptionKey       []byte
	FirestoreProjectID       string
	RateLimitPerMin          int
	SignupRateLimitPerHour   int
	AccessCodeMaxUses        int
	AccessCodeMaxFailures    int
	AccessCodeLockoutMinutes int
	AdminEmails              map[string]struct{}
	TurnstileSecretKey       string
	SignupAccessCodes        map[string]struct{}
}

// Load reads environment variables, applies safe defaults where appropriate,
// and validates required secrets/IDs before the app starts.
func Load() (Config, error) {
	cfg := Config{
		AppEnv:             getOrDefault("APP_ENV", "development"),
		Port:               getOrDefault("PORT", "8080"),
		AllowedOrigin:      getOrDefault("ALLOWED_ORIGIN", "http://localhost:3000"),
		StoreProvider:      strings.ToLower(getOrDefault("STORE_PROVIDER", "firestore")),
		SQLitePath:         getOrDefault("SQLITE_PATH", "/tmp/sync-audio-platforms.db"),
		SessionHMACKey:     os.Getenv("SESSION_HMAC_KEY"),
		FirestoreProjectID: os.Getenv("FIRESTORE_PROJECT_ID"),
		TurnstileSecretKey: strings.TrimSpace(os.Getenv("TURNSTILE_SECRET_KEY")),
		AdminEmails:        parseEmailSet(os.Getenv("ADMIN_EMAILS")),
		SignupAccessCodes:  parseCodeSet(os.Getenv("SIGNUP_ACCESS_CODES")),
	}

	// Session signing secret must be strong enough to resist brute-force attacks.
	if len(cfg.SessionHMACKey) < 32 {
		return Config{}, errors.New("SESSION_HMAC_KEY must be at least 32 characters")
	}
	if cfg.StoreProvider != "firestore" && cfg.StoreProvider != "sqlite" {
		return Config{}, errors.New("STORE_PROVIDER must be either firestore or sqlite")
	}
	if cfg.StoreProvider == "firestore" && cfg.FirestoreProjectID == "" {
		return Config{}, errors.New("FIRESTORE_PROJECT_ID is required when STORE_PROVIDER=firestore")
	}

	// Provider tokens are encrypted at rest with AES-256, so key must decode to 32 bytes.
	encKeyRaw := os.Getenv("TOKEN_ENCRYPTION_KEY")
	if encKeyRaw == "" {
		return Config{}, errors.New("TOKEN_ENCRYPTION_KEY is required")
	}
	decoded, err := base64.StdEncoding.DecodeString(encKeyRaw)
	if err != nil {
		return Config{}, fmt.Errorf("TOKEN_ENCRYPTION_KEY must be base64 encoded: %w", err)
	}
	if len(decoded) != 32 {
		return Config{}, errors.New("TOKEN_ENCRYPTION_KEY must decode to exactly 32 bytes")
	}
	cfg.TokenEncryptionKey = decoded

	// Basic abuse protection for public endpoints.
	rateLimit := getOrDefault("DEFAULT_RATE_LIMIT_PER_MIN", "120")
	cfg.RateLimitPerMin, err = strconv.Atoi(rateLimit)
	if err != nil || cfg.RateLimitPerMin <= 0 {
		return Config{}, errors.New("DEFAULT_RATE_LIMIT_PER_MIN must be a positive integer")
	}
	signupRateLimit := getOrDefault("SIGNUP_RATE_LIMIT_PER_HOUR", "10")
	cfg.SignupRateLimitPerHour, err = strconv.Atoi(signupRateLimit)
	if err != nil || cfg.SignupRateLimitPerHour <= 0 {
		return Config{}, errors.New("SIGNUP_RATE_LIMIT_PER_HOUR must be a positive integer")
	}
	codeMaxUses := getOrDefault("ACCESS_CODE_MAX_USES", "1")
	cfg.AccessCodeMaxUses, err = strconv.Atoi(codeMaxUses)
	if err != nil || cfg.AccessCodeMaxUses <= 0 {
		return Config{}, errors.New("ACCESS_CODE_MAX_USES must be a positive integer")
	}
	codeMaxFailures := getOrDefault("ACCESS_CODE_MAX_FAILURES", "5")
	cfg.AccessCodeMaxFailures, err = strconv.Atoi(codeMaxFailures)
	if err != nil || cfg.AccessCodeMaxFailures <= 0 {
		return Config{}, errors.New("ACCESS_CODE_MAX_FAILURES must be a positive integer")
	}
	codeLockoutMinutes := getOrDefault("ACCESS_CODE_LOCKOUT_MINUTES", "15")
	cfg.AccessCodeLockoutMinutes, err = strconv.Atoi(codeLockoutMinutes)
	if err != nil || cfg.AccessCodeLockoutMinutes <= 0 {
		return Config{}, errors.New("ACCESS_CODE_LOCKOUT_MINUTES must be a positive integer")
	}

	return cfg, nil
}

func getOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func parseEmailSet(raw string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, piece := range strings.Split(raw, ",") {
		email := strings.ToLower(strings.TrimSpace(piece))
		if email == "" {
			continue
		}
		out[email] = struct{}{}
	}
	return out
}

func parseCodeSet(raw string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, piece := range strings.Split(raw, ",") {
		code := strings.TrimSpace(piece)
		if code == "" {
			continue
		}
		out[code] = struct{}{}
	}
	return out
}
