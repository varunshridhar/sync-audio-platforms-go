package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/varun/sync-audio-platforms-go/backend/internal/domain"
)

type FirestoreStore struct {
	client *firestore.Client
}

// NewFirestoreStore creates the Firestore client used for all DB operations.
func NewFirestoreStore(ctx context.Context, projectID string) (*FirestoreStore, error) {
	c, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &FirestoreStore{client: c}, nil
}

func (s *FirestoreStore) Close() error {
	return s.client.Close()
}

// CreateOrGetUserByEmail performs lookup first, then creates a new user if absent.
func (s *FirestoreStore) CreateOrGetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	iter := s.client.Collection("users").Where("email", "==", email).Limit(1).Documents(ctx)
	doc, err := iter.Next()
	if err == nil {
		var u domain.User
		if err := doc.DataTo(&u); err != nil {
			return domain.User{}, err
		}
		if u.Status == "" {
			// Backwards compatibility for users created before approval workflow.
			u.Status = domain.UserStatusApproved
			if _, err := s.client.Collection("users").Doc(u.ID).Set(ctx, map[string]any{
				"status": u.Status,
			}, firestore.MergeAll); err != nil {
				return domain.User{}, err
			}
		}
		return u, nil
	}
	if !errors.Is(err, iterator.Done) {
		return domain.User{}, err
	}

	// If not found, create a new user document with UUID.
	u := domain.User{
		ID:        uuid.NewString(),
		Email:     email,
		Status:    domain.UserStatusPending,
		CreatedAt: time.Now().UTC(),
	}
	_, err = s.client.Collection("users").Doc(u.ID).Set(ctx, u)
	return u, err
}

// GetUserByID reads one user document by user ID.
func (s *FirestoreStore) GetUserByID(ctx context.Context, id string) (domain.User, error) {
	doc, err := s.client.Collection("users").Doc(id).Get(ctx)
	if err != nil {
		return domain.User{}, err
	}
	var u domain.User
	if err := doc.DataTo(&u); err != nil {
		return domain.User{}, err
	}
	if u.Status == "" {
		// Backwards compatibility for users created before approval workflow.
		u.Status = domain.UserStatusApproved
	}
	return u, nil
}

func (s *FirestoreStore) ListUsersByStatus(ctx context.Context, status domain.UserStatus, limit int) ([]domain.User, error) {
	if limit <= 0 {
		limit = 50
	}
	iter := s.client.Collection("users").
		Where("status", "==", status).
		OrderBy("createdAt", firestore.Asc).
		Limit(limit).
		Documents(ctx)
	defer iter.Stop()

	users := make([]domain.User, 0, limit)
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var u domain.User
		if err := doc.DataTo(&u); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (s *FirestoreStore) ApproveUser(ctx context.Context, id, approvedBy string) (domain.User, error) {
	return s.SetUserStatus(ctx, id, domain.UserStatusApproved, approvedBy)
}

func (s *FirestoreStore) SetUserStatus(ctx context.Context, id string, status domain.UserStatus, approvedBy string) (domain.User, error) {
	now := time.Now().UTC()
	update := map[string]any{
		"status": status,
	}
	if status == domain.UserStatusApproved {
		update["approvedBy"] = approvedBy
		update["approvedAt"] = now
	}
	_, err := s.client.Collection("users").Doc(id).Set(ctx, update, firestore.MergeAll)
	if err != nil {
		return domain.User{}, err
	}
	return s.GetUserByID(ctx, id)
}

// RedeemSignupAccessCode atomically enforces global max uses and one redemption per user.
func (s *FirestoreStore) RedeemSignupAccessCode(ctx context.Context, code, userID string, maxUses int) (bool, error) {
	if maxUses <= 0 {
		maxUses = 1
	}
	codeHash := sha256.Sum256([]byte(code))
	codeDocID := hex.EncodeToString(codeHash[:])
	usageRef := s.client.Collection("access_code_usage").Doc(codeDocID)
	redeemRef := usageRef.Collection("redemptions").Doc(userID)

	redeemed := false
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		if _, err := tx.Get(redeemRef); err == nil {
			// Idempotent path: user already redeemed this code before.
			redeemed = true
			return nil
		} else if status.Code(err) != codes.NotFound {
			return err
		}

		var count int64
		usageDoc, err := tx.Get(usageRef)
		if err != nil {
			if status.Code(err) != codes.NotFound {
				return err
			}
			count = 0
		} else {
			if raw, ok := usageDoc.Data()["count"]; ok {
				switch v := raw.(type) {
				case int64:
					count = v
				case int:
					count = int64(v)
				case float64:
					count = int64(v)
				}
			}
		}

		if count >= int64(maxUses) {
			redeemed = false
			return nil
		}

		now := time.Now().UTC()
		if err := tx.Set(redeemRef, map[string]any{
			"userId": userID,
			"usedAt": now,
		}); err != nil {
			return err
		}
		if err := tx.Set(usageRef, map[string]any{
			"count":     count + 1,
			"updatedAt": now,
		}, firestore.MergeAll); err != nil {
			return err
		}
		redeemed = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return redeemed, nil
}

// UpsertConnectedAccount creates or updates the user-provider account document.
func (s *FirestoreStore) UpsertConnectedAccount(ctx context.Context, account domain.ConnectedAccount) error {
	docID := account.UserID + "_" + string(account.Provider)
	_, err := s.client.Collection("connected_accounts").Doc(docID).Set(ctx, account)
	return err
}

// ListConnectedAccounts returns all provider connections for a given user.
func (s *FirestoreStore) ListConnectedAccounts(ctx context.Context, userID string) ([]domain.ConnectedAccount, error) {
	iter := s.client.Collection("connected_accounts").Where("userId", "==", userID).Documents(ctx)
	defer iter.Stop()

	results := make([]domain.ConnectedAccount, 0)
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var account domain.ConnectedAccount
		if err := doc.DataTo(&account); err != nil {
			return nil, err
		}
		results = append(results, account)
	}
	return results, nil
}

// CreateSyncJob writes a new sync job in "pending" state unless caller set a status.
func (s *FirestoreStore) CreateSyncJob(ctx context.Context, job domain.SyncJob) (domain.SyncJob, error) {
	if job.ID == "" {
		job.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	job.CreatedAt = now
	job.UpdatedAt = now
	if job.Status == "" {
		job.Status = domain.SyncJobPending
	}
	_, err := s.client.Collection("sync_jobs").Doc(job.ID).Set(ctx, job)
	return job, err
}

// ListSyncJobs returns newest jobs first so UI can show recent status at top.
func (s *FirestoreStore) ListSyncJobs(ctx context.Context, userID string, limit int) ([]domain.SyncJob, error) {
	if limit <= 0 {
		limit = 20
	}
	iter := s.client.Collection("sync_jobs").
		Where("userId", "==", userID).
		OrderBy("createdAt", firestore.Desc).
		Limit(limit).
		Documents(ctx)
	defer iter.Stop()

	out := make([]domain.SyncJob, 0, limit)
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var job domain.SyncJob
		if err := doc.DataTo(&job); err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	return out, nil
}

// UpdateSyncJobStatus updates status/error/timestamp atomically using merge.
func (s *FirestoreStore) UpdateSyncJobStatus(ctx context.Context, jobID string, status domain.SyncJobStatus, errMsg string) error {
	_, err := s.client.Collection("sync_jobs").Doc(jobID).Set(ctx, map[string]any{
		"status":    status,
		"error":     errMsg,
		"updatedAt": time.Now().UTC(),
	}, firestore.MergeAll)
	return err
}
