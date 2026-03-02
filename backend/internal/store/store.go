package store

import (
	"context"

	"github.com/varun/sync-audio-platforms-go/backend/internal/domain"
)

// Store is the persistence contract used by handlers/services.
// Keeping this as an interface makes it easy to swap Firestore for another DB later.
type Store interface {
	CreateOrGetUserByEmail(ctx context.Context, email string) (domain.User, error)
	GetUserByID(ctx context.Context, id string) (domain.User, error)
	ListUsersByStatus(ctx context.Context, status domain.UserStatus, limit int) ([]domain.User, error)
	ApproveUser(ctx context.Context, id, approvedBy string) (domain.User, error)
	SetUserStatus(ctx context.Context, id string, status domain.UserStatus, approvedBy string) (domain.User, error)
	RedeemSignupAccessCode(ctx context.Context, code, userID string, maxUses int) (bool, error)
	UpsertConnectedAccount(ctx context.Context, account domain.ConnectedAccount) error
	ListConnectedAccounts(ctx context.Context, userID string) ([]domain.ConnectedAccount, error)
	CreateSyncJob(ctx context.Context, job domain.SyncJob) (domain.SyncJob, error)
	ListSyncJobs(ctx context.Context, userID string, limit int) ([]domain.SyncJob, error)
	UpdateSyncJobStatus(ctx context.Context, jobID string, status domain.SyncJobStatus, errMsg string) error
	Close() error
}
