package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/varun/sync-audio-platforms-go/backend/internal/config"
	"github.com/varun/sync-audio-platforms-go/backend/internal/domain"
	"github.com/varun/sync-audio-platforms-go/backend/internal/httpx"
	"github.com/varun/sync-audio-platforms-go/backend/internal/security"
	"github.com/varun/sync-audio-platforms-go/backend/internal/store"
)

type App struct {
	cfg             config.Config
	store           store.Store
	sessionManager  *security.SessionManager
	cipher          *security.Cipher
	signupLimiter   *ipWindowLimiter
	accessCodeGuard *accessCodeAbuseGuard
}

// New wires all runtime dependencies (store, crypto, session manager).
func New(cfg config.Config) (*App, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var appStore store.Store
	switch cfg.StoreProvider {
	case "sqlite":
		sqliteStore, err := store.NewSQLiteStore(cfg.SQLitePath)
		if err != nil {
			return nil, err
		}
		appStore = sqliteStore
	case "firestore":
		firestoreStore, err := store.NewFirestoreStore(ctx, cfg.FirestoreProjectID)
		if err != nil {
			return nil, err
		}
		appStore = firestoreStore
	default:
		return nil, errors.New("unsupported store provider")
	}
	c, err := security.NewCipher(cfg.TokenEncryptionKey)
	if err != nil {
		return nil, err
	}

	return &App{
		cfg:            cfg,
		store:          appStore,
		sessionManager: security.NewSessionManager(cfg.SessionHMACKey),
		cipher:         c,
		signupLimiter:  newIPWindowLimiter(time.Hour, cfg.SignupRateLimitPerHour),
		accessCodeGuard: newAccessCodeAbuseGuard(
			time.Hour,
			cfg.AccessCodeMaxFailures,
			time.Duration(cfg.AccessCodeLockoutMinutes)*time.Minute,
		),
	}, nil
}

func (a *App) Close() error {
	return a.store.Close()
}

// Router registers all API endpoints and wraps them with global middleware.
func (a *App) Router() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/health", a.handleHealth)
	mux.HandleFunc("GET /v1/docs", a.handleDocs)
	mux.HandleFunc("GET /v1/docs/openapi.yaml", a.handleOpenAPISpec)
	mux.HandleFunc("POST /v1/auth/login", a.handleLogin)
	mux.HandleFunc("POST /v1/auth/logout", a.authRequired(a.handleLogout))
	mux.HandleFunc("GET /v1/me", a.authRequired(a.handleMe))
	mux.HandleFunc("POST /v1/providers/connect", a.authRequired(a.approvedRequired(a.handleConnectProvider)))
	mux.HandleFunc("GET /v1/providers", a.authRequired(a.approvedRequired(a.handleListProviders)))
	mux.HandleFunc("POST /v1/sync/jobs", a.authRequired(a.approvedRequired(a.handleCreateSyncJob)))
	mux.HandleFunc("GET /v1/sync/jobs", a.authRequired(a.approvedRequired(a.handleListSyncJobs)))
	mux.HandleFunc("GET /v1/admin/users/pending", a.authRequired(a.adminRequired(a.handleListPendingUsers)))
	mux.HandleFunc("POST /v1/admin/users/{userID}/approve", a.authRequired(a.adminRequired(a.handleApproveUser)))

	// Middleware order matters: outer wrappers execute first on request.
	chain := httpx.Recover(
		httpx.RequestLogger(
			httpx.SecurityHeaders(
				httpx.CORS(a.cfg.AllowedOrigin)(
					httpx.Timeout(10 * time.Second)(
						httpx.RateLimit(a.cfg.RateLimitPerMin)(mux),
					),
				),
			),
		),
	)

	return chain
}

func validateProvider(provider domain.Provider) bool {
	return provider == domain.ProviderSpotify || provider == domain.ProviderYouTubeMusic
}
