package app

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/varun/sync-audio-platforms-go/backend/internal/domain"
	"github.com/varun/sync-audio-platforms-go/backend/internal/httpx"
)

var emailRx = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// handleHealth is used for liveness/readiness checks.
func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type loginRequest struct {
	Email        string `json:"email"`
	CaptchaToken string `json:"captchaToken"`
	Website      string `json:"website"`
	AccessCode   string `json:"accessCode"`
}

// handleLogin creates or finds user, then sets a signed session cookie.
func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !emailRx.MatchString(email) {
		httpx.Error(w, http.StatusBadRequest, "invalid email")
		return
	}
	if strings.TrimSpace(req.Website) != "" {
		httpx.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	clientIP := httpx.ClientIP(r)
	if !a.signupLimiter.allow(clientIP, time.Now().UTC()) {
		httpx.Error(w, http.StatusTooManyRequests, "too many signup attempts")
		return
	}
	if a.cfg.TurnstileSecretKey != "" {
		ok, err := verifyTurnstile(r.Context(), a.cfg.TurnstileSecretKey, strings.TrimSpace(req.CaptchaToken), clientIP)
		if err != nil {
			httpx.Error(w, http.StatusBadGateway, "failed to verify captcha")
			return
		}
		if !ok {
			httpx.Error(w, http.StatusBadRequest, "captcha verification failed")
			return
		}
	}

	user, err := a.store.CreateOrGetUserByEmail(r.Context(), email)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to login")
		return
	}
	accessCode := strings.TrimSpace(req.AccessCode)
	if accessCode != "" && user.Status != domain.UserStatusApproved {
		accessCodeKey := clientIP + "|" + email
		if a.accessCodeGuard.isLocked(accessCodeKey, time.Now().UTC()) {
			httpx.Error(w, http.StatusTooManyRequests, "too many invalid access code attempts; try again later")
			return
		}
		if _, ok := a.cfg.SignupAccessCodes[accessCode]; !ok {
			if a.accessCodeGuard.registerFailure(accessCodeKey, time.Now().UTC()) {
				httpx.Error(w, http.StatusTooManyRequests, "too many invalid access code attempts; try again later")
				return
			}
			httpx.Error(w, http.StatusBadRequest, "invalid access code")
			return
		}
		redeemed, err := a.store.RedeemSignupAccessCode(r.Context(), accessCode, user.ID, a.cfg.AccessCodeMaxUses)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "failed to redeem access code")
			return
		}
		if !redeemed {
			if a.accessCodeGuard.registerFailure(accessCodeKey, time.Now().UTC()) {
				httpx.Error(w, http.StatusTooManyRequests, "too many invalid access code attempts; try again later")
				return
			}
			httpx.Error(w, http.StatusForbidden, "access code has reached its usage limit")
			return
		}
		user, err = a.store.SetUserStatus(r.Context(), user.ID, domain.UserStatusApproved, "access_code")
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "failed to apply access code")
			return
		}
		a.accessCodeGuard.clearFailures(accessCodeKey)
	}

	token, err := a.sessionManager.Sign(user.ID, 24*time.Hour)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	// Secure=true is enabled only in production HTTPS environments.
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.cfg.AppEnv == "production",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int((24 * time.Hour).Seconds()),
	})

	httpx.JSON(w, http.StatusOK, user)
}

// handleLogout clears the session cookie in browser.
func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   a.cfg.AppEnv == "production",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleMe returns the currently authenticated user's profile.
func (a *App) handleMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.UserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	user, err := a.store.GetUserByID(r.Context(), userID)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "user not found")
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

func (a *App) handleListPendingUsers(w http.ResponseWriter, r *http.Request) {
	users, err := a.store.ListUsersByStatus(r.Context(), domain.UserStatusPending, 100)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list pending users")
		return
	}
	httpx.JSON(w, http.StatusOK, users)
}

func (a *App) handleApproveUser(w http.ResponseWriter, r *http.Request) {
	adminID, ok := httpx.UserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	targetUserID := strings.TrimSpace(r.PathValue("userID"))
	if targetUserID == "" {
		httpx.Error(w, http.StatusBadRequest, "userID is required")
		return
	}
	approvedUser, err := a.store.ApproveUser(r.Context(), targetUserID, adminID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to approve user")
		return
	}
	httpx.JSON(w, http.StatusOK, approvedUser)
}

type connectProviderRequest struct {
	Provider     domain.Provider `json:"provider"`
	AccessToken  string          `json:"accessToken"`
	RefreshToken string          `json:"refreshToken"`
	ExpiryUnix   int64           `json:"expiryUnix"`
}

// handleConnectProvider stores encrypted provider tokens for a user.
func (a *App) handleConnectProvider(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.UserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req connectProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if !validateProvider(req.Provider) {
		httpx.Error(w, http.StatusBadRequest, "unsupported provider")
		return
	}
	if strings.TrimSpace(req.AccessToken) == "" || strings.TrimSpace(req.RefreshToken) == "" {
		httpx.Error(w, http.StatusBadRequest, "provider tokens are required")
		return
	}

	// Never store raw provider credentials; always encrypt before persistence.
	encAccess, err := a.cipher.Encrypt(req.AccessToken)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to encrypt credentials")
		return
	}
	encRefresh, err := a.cipher.Encrypt(req.RefreshToken)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to encrypt credentials")
		return
	}

	account := domain.ConnectedAccount{
		UserID:              userID,
		Provider:            req.Provider,
		EncryptedAccess:     encAccess,
		EncryptedRefresh:    encRefresh,
		TokenExpiryUnix:     req.ExpiryUnix,
		LastSyncCheckpoint:  "",
		ConnectedAt:         time.Now().UTC(),
		LastRotationUnixSec: time.Now().UTC().Unix(),
	}
	if err := a.store.UpsertConnectedAccount(r.Context(), account); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to connect provider")
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"userId":   userID,
		"provider": req.Provider,
		"status":   "connected",
	})
}

// handleListProviders returns safe provider metadata (without credentials).
func (a *App) handleListProviders(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.UserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	accounts, err := a.store.ListConnectedAccounts(r.Context(), userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list providers")
		return
	}
	type safeAccount struct {
		UserID             string          `json:"userId"`
		Provider           domain.Provider `json:"provider"`
		TokenExpiryUnix    int64           `json:"tokenExpiryUnix"`
		LastSyncCheckpoint string          `json:"lastSyncCheckpoint"`
		ConnectedAt        time.Time       `json:"connectedAt"`
	}
	resp := make([]safeAccount, 0, len(accounts))
	for _, a := range accounts {
		resp = append(resp, safeAccount{
			UserID:             a.UserID,
			Provider:           a.Provider,
			TokenExpiryUnix:    a.TokenExpiryUnix,
			LastSyncCheckpoint: a.LastSyncCheckpoint,
			ConnectedAt:        a.ConnectedAt,
		})
	}
	httpx.JSON(w, http.StatusOK, resp)
}

type createSyncJobRequest struct {
	Source      domain.Provider `json:"source"`
	Destination domain.Provider `json:"destination"`
	PlaylistID  string          `json:"playlistId"`
}

// handleCreateSyncJob validates input and creates a new sync task record.
func (a *App) handleCreateSyncJob(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.UserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req createSyncJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if !validateProvider(req.Source) || !validateProvider(req.Destination) || req.Source == req.Destination {
		httpx.Error(w, http.StatusBadRequest, "invalid source/destination providers")
		return
	}
	if strings.TrimSpace(req.PlaylistID) == "" {
		httpx.Error(w, http.StatusBadRequest, "playlistId is required")
		return
	}

	job := domain.SyncJob{
		UserID:      userID,
		Source:      req.Source,
		Destination: req.Destination,
		PlaylistID:  req.PlaylistID,
		Status:      domain.SyncJobPending,
	}
	job, err := a.store.CreateSyncJob(r.Context(), job)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to create sync job")
		return
	}

	// TODO: enqueue Cloud Tasks job payload with job.ID for async processing.
	httpx.JSON(w, http.StatusCreated, job)
}

// handleListSyncJobs returns recent sync jobs for dashboard status display.
func (a *App) handleListSyncJobs(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.UserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	jobs, err := a.store.ListSyncJobs(r.Context(), userID, 30)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list jobs")
		return
	}
	httpx.JSON(w, http.StatusOK, jobs)
}

// authRequired is route middleware that validates session cookie
// and injects user ID into request context.
func (a *App) authRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, "missing session")
			return
		}
		claims, err := a.sessionManager.Verify(cookie.Value)
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, "invalid session")
			return
		}
		next(w, httpx.WithUserID(r, claims.UserID))
	}
}

func (a *App) approvedRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := httpx.UserIDFromContext(r.Context())
		if !ok {
			httpx.Error(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		user, err := a.store.GetUserByID(r.Context(), userID)
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, "user not found")
			return
		}
		if user.Status != domain.UserStatusApproved {
			httpx.Error(w, http.StatusForbidden, "account pending developer approval")
			return
		}
		next(w, r)
	}
}

func (a *App) adminRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := httpx.UserIDFromContext(r.Context())
		if !ok {
			httpx.Error(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		user, err := a.store.GetUserByID(r.Context(), userID)
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, "user not found")
			return
		}
		if _, exists := a.cfg.AdminEmails[strings.ToLower(strings.TrimSpace(user.Email))]; !exists {
			httpx.Error(w, http.StatusForbidden, "admin access required")
			return
		}
		next(w, r)
	}
}
