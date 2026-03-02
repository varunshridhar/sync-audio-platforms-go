package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/varun/sync-audio-platforms-go/backend/internal/config"
	"github.com/varun/sync-audio-platforms-go/backend/internal/domain"
	"github.com/varun/sync-audio-platforms-go/backend/internal/security"
)

type mockStore struct {
	user domain.User
}

func (m *mockStore) CreateOrGetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	u := m.user
	u.Email = email
	if u.ID == "" {
		u.ID = "u-1"
	}
	return u, nil
}

func (m *mockStore) GetUserByID(ctx context.Context, id string) (domain.User, error) {
	u := m.user
	u.ID = id
	return u, nil
}

func (m *mockStore) ListUsersByStatus(ctx context.Context, status domain.UserStatus, limit int) ([]domain.User, error) {
	return nil, nil
}

func (m *mockStore) ApproveUser(ctx context.Context, id, approvedBy string) (domain.User, error) {
	return domain.User{ID: id, Status: domain.UserStatusApproved}, nil
}

func (m *mockStore) SetUserStatus(ctx context.Context, id string, status domain.UserStatus, approvedBy string) (domain.User, error) {
	return domain.User{ID: id, Status: status}, nil
}

func (m *mockStore) RedeemSignupAccessCode(ctx context.Context, code, userID string, maxUses int) (bool, error) {
	return false, nil
}

func (m *mockStore) UpsertConnectedAccount(ctx context.Context, account domain.ConnectedAccount) error {
	return nil
}

func (m *mockStore) ListConnectedAccounts(ctx context.Context, userID string) ([]domain.ConnectedAccount, error) {
	return nil, nil
}

func (m *mockStore) CreateSyncJob(ctx context.Context, job domain.SyncJob) (domain.SyncJob, error) {
	return job, nil
}

func (m *mockStore) ListSyncJobs(ctx context.Context, userID string, limit int) ([]domain.SyncJob, error) {
	return nil, nil
}

func (m *mockStore) UpdateSyncJobStatus(ctx context.Context, jobID string, status domain.SyncJobStatus, errMsg string) error {
	return nil
}

func (m *mockStore) Close() error { return nil }

func TestDocsEndpoints(t *testing.T) {
	a := &App{
		cfg: config.Config{
			AllowedOrigin:   "http://localhost:3000",
			RateLimitPerMin: 120,
		},
	}
	router := a.Router()

	reqDocs := httptest.NewRequest(http.MethodGet, "/v1/docs", nil)
	recDocs := httptest.NewRecorder()
	router.ServeHTTP(recDocs, reqDocs)
	if recDocs.Code != http.StatusOK {
		t.Fatalf("docs endpoint status = %d, want 200", recDocs.Code)
	}
	if !bytes.Contains(recDocs.Body.Bytes(), []byte("swagger-ui")) {
		t.Fatalf("docs endpoint response should include swagger ui container")
	}

	reqSpec := httptest.NewRequest(http.MethodGet, "/v1/docs/openapi.yaml", nil)
	recSpec := httptest.NewRecorder()
	router.ServeHTTP(recSpec, reqSpec)
	if recSpec.Code != http.StatusOK {
		t.Fatalf("openapi endpoint status = %d, want 200", recSpec.Code)
	}
	if !bytes.Contains(recSpec.Body.Bytes(), []byte("openapi: 3.0.3")) {
		t.Fatalf("openapi spec response missing expected header")
	}
}

func TestHandleLogin_AccessCodeLockout(t *testing.T) {
	store := &mockStore{
		user: domain.User{
			ID:     "u-1",
			Status: domain.UserStatusPending,
		},
	}
	a := &App{
		cfg: config.Config{
			SignupAccessCodes: map[string]struct{}{
				"Ryanthisisforoouuuuu": {},
			},
			AccessCodeMaxUses: 1,
		},
		store:           store,
		sessionManager:  security.NewSessionManager("0123456789abcdef0123456789abcdef"),
		signupLimiter:   newIPWindowLimiter(time.Hour, 100),
		accessCodeGuard: newAccessCodeAbuseGuard(time.Hour, 2, 15*time.Minute),
	}

	makeReq := func() *http.Request {
		body, _ := json.Marshal(map[string]any{
			"email":      "user@example.com",
			"accessCode": "wrong-code",
		})
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(body))
		req.RemoteAddr = "127.0.0.1:12345"
		return req
	}

	rec1 := httptest.NewRecorder()
	a.handleLogin(rec1, makeReq())
	if rec1.Code != http.StatusBadRequest {
		t.Fatalf("first invalid code status = %d, want 400", rec1.Code)
	}

	rec2 := httptest.NewRecorder()
	a.handleLogin(rec2, makeReq())
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("second invalid code status = %d, want 429", rec2.Code)
	}
}
