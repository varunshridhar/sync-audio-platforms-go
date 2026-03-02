package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"github.com/varun/sync-audio-platforms-go/backend/internal/domain"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	store := &SQLiteStore{db: db}
	if err := store.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) initSchema() error {
	schema := `
PRAGMA journal_mode=WAL;
CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL,
  approved_by TEXT,
  approved_at TEXT,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS connected_accounts (
  user_id TEXT NOT NULL,
  provider TEXT NOT NULL,
  encrypted_access TEXT NOT NULL,
  encrypted_refresh TEXT NOT NULL,
  token_expiry_unix INTEGER NOT NULL,
  last_sync_checkpoint TEXT NOT NULL,
  connected_at TEXT NOT NULL,
  last_rotation_unix_sec INTEGER NOT NULL,
  PRIMARY KEY (user_id, provider)
);
CREATE TABLE IF NOT EXISTS sync_jobs (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  source TEXT NOT NULL,
  destination TEXT NOT NULL,
  playlist_id TEXT NOT NULL,
  status TEXT NOT NULL,
  error TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sync_jobs_user_created_at ON sync_jobs(user_id, created_at DESC);
CREATE TABLE IF NOT EXISTS access_code_usage (
  code_hash TEXT PRIMARY KEY,
  count INTEGER NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS access_code_redemptions (
  code_hash TEXT NOT NULL,
  user_id TEXT NOT NULL,
  used_at TEXT NOT NULL,
  PRIMARY KEY (code_hash, user_id)
);`
	_, err := s.db.Exec(schema)
	return err
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

func (s *SQLiteStore) CreateOrGetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	query := `SELECT id,email,status,approved_by,approved_at,created_at FROM users WHERE email = ? LIMIT 1`
	row := s.db.QueryRowContext(ctx, query, email)
	user, err := scanUser(row)
	if err == nil {
		if user.Status == "" {
			user.Status = domain.UserStatusApproved
			_, _ = s.db.ExecContext(ctx, `UPDATE users SET status = ? WHERE id = ?`, string(user.Status), user.ID)
		}
		return user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, err
	}

	now := time.Now().UTC()
	user = domain.User{
		ID:        uuid.NewString(),
		Email:     email,
		Status:    domain.UserStatusPending,
		CreatedAt: now,
	}
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO users(id,email,status,approved_by,approved_at,created_at) VALUES(?,?,?,?,?,?)`,
		user.ID, user.Email, string(user.Status), "", nil, now.Format(time.RFC3339Nano),
	)
	return user, err
}

func (s *SQLiteStore) GetUserByID(ctx context.Context, id string) (domain.User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id,email,status,approved_by,approved_at,created_at FROM users WHERE id = ?`, id)
	user, err := scanUser(row)
	if err != nil {
		return domain.User{}, err
	}
	if user.Status == "" {
		user.Status = domain.UserStatusApproved
	}
	return user, nil
}

func (s *SQLiteStore) ListUsersByStatus(ctx context.Context, status domain.UserStatus, limit int) ([]domain.User, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id,email,status,approved_by,approved_at,created_at FROM users WHERE status = ? ORDER BY created_at ASC LIMIT ?`,
		string(status), limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.User, 0, limit)
	for rows.Next() {
		user, err := scanUserFromRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, user)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ApproveUser(ctx context.Context, id, approvedBy string) (domain.User, error) {
	return s.SetUserStatus(ctx, id, domain.UserStatusApproved, approvedBy)
}

func (s *SQLiteStore) SetUserStatus(ctx context.Context, id string, status domain.UserStatus, approvedBy string) (domain.User, error) {
	var approvedAt any = nil
	if status == domain.UserStatusApproved {
		approvedAt = time.Now().UTC().Format(time.RFC3339Nano)
	} else {
		approvedBy = ""
	}
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE users SET status = ?, approved_by = ?, approved_at = ? WHERE id = ?`,
		string(status), approvedBy, approvedAt, id,
	)
	if err != nil {
		return domain.User{}, err
	}
	return s.GetUserByID(ctx, id)
}

func (s *SQLiteStore) RedeemSignupAccessCode(ctx context.Context, code, userID string, maxUses int) (bool, error) {
	if maxUses <= 0 {
		maxUses = 1
	}
	hash := sha256.Sum256([]byte(code))
	codeHash := hex.EncodeToString(hash[:])
	now := time.Now().UTC().Format(time.RFC3339Nano)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var existing string
	err = tx.QueryRowContext(
		ctx,
		`SELECT used_at FROM access_code_redemptions WHERE code_hash = ? AND user_id = ?`,
		codeHash, userID,
	).Scan(&existing)
	if err == nil {
		if err := tx.Commit(); err != nil {
			return false, err
		}
		return true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}

	var count int
	err = tx.QueryRowContext(ctx, `SELECT count FROM access_code_usage WHERE code_hash = ?`, codeHash).Scan(&count)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	if errors.Is(err, sql.ErrNoRows) {
		count = 0
	}
	if count >= maxUses {
		if err := tx.Commit(); err != nil {
			return false, err
		}
		return false, nil
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO access_code_redemptions(code_hash,user_id,used_at) VALUES(?,?,?)`,
		codeHash, userID, now,
	); err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO access_code_usage(code_hash,count,updated_at) VALUES(?,?,?)
		 ON CONFLICT(code_hash) DO UPDATE SET count = excluded.count, updated_at = excluded.updated_at`,
		codeHash, count+1, now,
	); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *SQLiteStore) UpsertConnectedAccount(ctx context.Context, account domain.ConnectedAccount) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO connected_accounts(
		 user_id,provider,encrypted_access,encrypted_refresh,token_expiry_unix,last_sync_checkpoint,connected_at,last_rotation_unix_sec
		) VALUES(?,?,?,?,?,?,?,?)
		ON CONFLICT(user_id,provider) DO UPDATE SET
		  encrypted_access=excluded.encrypted_access,
		  encrypted_refresh=excluded.encrypted_refresh,
		  token_expiry_unix=excluded.token_expiry_unix,
		  last_sync_checkpoint=excluded.last_sync_checkpoint,
		  connected_at=excluded.connected_at,
		  last_rotation_unix_sec=excluded.last_rotation_unix_sec`,
		account.UserID, string(account.Provider), account.EncryptedAccess, account.EncryptedRefresh,
		account.TokenExpiryUnix, account.LastSyncCheckpoint, account.ConnectedAt.Format(time.RFC3339Nano), account.LastRotationUnixSec,
	)
	return err
}

func (s *SQLiteStore) ListConnectedAccounts(ctx context.Context, userID string) ([]domain.ConnectedAccount, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT user_id,provider,encrypted_access,encrypted_refresh,token_expiry_unix,last_sync_checkpoint,connected_at,last_rotation_unix_sec
		 FROM connected_accounts WHERE user_id = ?`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.ConnectedAccount, 0)
	for rows.Next() {
		var (
			account        domain.ConnectedAccount
			provider       string
			connectedAtRaw string
		)
		if err := rows.Scan(
			&account.UserID, &provider, &account.EncryptedAccess, &account.EncryptedRefresh,
			&account.TokenExpiryUnix, &account.LastSyncCheckpoint, &connectedAtRaw, &account.LastRotationUnixSec,
		); err != nil {
			return nil, err
		}
		account.Provider = domain.Provider(provider)
		account.ConnectedAt, err = time.Parse(time.RFC3339Nano, connectedAtRaw)
		if err != nil {
			return nil, err
		}
		out = append(out, account)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) CreateSyncJob(ctx context.Context, job domain.SyncJob) (domain.SyncJob, error) {
	if job.ID == "" {
		job.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	job.CreatedAt = now
	job.UpdatedAt = now
	if job.Status == "" {
		job.Status = domain.SyncJobPending
	}
	if job.Error == "" {
		job.Error = ""
	}
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO sync_jobs(id,user_id,source,destination,playlist_id,status,error,created_at,updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?)`,
		job.ID, job.UserID, string(job.Source), string(job.Destination), job.PlaylistID, string(job.Status),
		job.Error, job.CreatedAt.Format(time.RFC3339Nano), job.UpdatedAt.Format(time.RFC3339Nano),
	)
	return job, err
}

func (s *SQLiteStore) ListSyncJobs(ctx context.Context, userID string, limit int) ([]domain.SyncJob, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id,user_id,source,destination,playlist_id,status,error,created_at,updated_at
		 FROM sync_jobs WHERE user_id = ? ORDER BY created_at DESC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.SyncJob, 0, limit)
	for rows.Next() {
		var (
			job            domain.SyncJob
			sourceRaw      string
			destinationRaw string
			statusRaw      string
			createdAtRaw   string
			updatedAtRaw   string
		)
		if err := rows.Scan(
			&job.ID, &job.UserID, &sourceRaw, &destinationRaw, &job.PlaylistID, &statusRaw, &job.Error, &createdAtRaw, &updatedAtRaw,
		); err != nil {
			return nil, err
		}
		job.Source = domain.Provider(sourceRaw)
		job.Destination = domain.Provider(destinationRaw)
		job.Status = domain.SyncJobStatus(statusRaw)
		job.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAtRaw)
		if err != nil {
			return nil, err
		}
		job.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAtRaw)
		if err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) UpdateSyncJobStatus(ctx context.Context, jobID string, status domain.SyncJobStatus, errMsg string) error {
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE sync_jobs SET status = ?, error = ?, updated_at = ? WHERE id = ?`,
		string(status), errMsg, time.Now().UTC().Format(time.RFC3339Nano), jobID,
	)
	return err
}

func scanUser(row *sql.Row) (domain.User, error) {
	var (
		user          domain.User
		statusRaw     string
		approvedByRaw sql.NullString
		approvedAtRaw sql.NullString
		createdAtRaw  string
	)
	if err := row.Scan(&user.ID, &user.Email, &statusRaw, &approvedByRaw, &approvedAtRaw, &createdAtRaw); err != nil {
		return domain.User{}, err
	}
	user.Status = domain.UserStatus(statusRaw)
	if approvedByRaw.Valid {
		user.ApprovedBy = approvedByRaw.String
	}
	if approvedAtRaw.Valid && approvedAtRaw.String != "" {
		t, err := time.Parse(time.RFC3339Nano, approvedAtRaw.String)
		if err != nil {
			return domain.User{}, err
		}
		user.ApprovedAt = &t
	}
	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return domain.User{}, err
	}
	user.CreatedAt = createdAt
	return user, nil
}

func scanUserFromRows(rows *sql.Rows) (domain.User, error) {
	var (
		user          domain.User
		statusRaw     string
		approvedByRaw sql.NullString
		approvedAtRaw sql.NullString
		createdAtRaw  string
	)
	if err := rows.Scan(&user.ID, &user.Email, &statusRaw, &approvedByRaw, &approvedAtRaw, &createdAtRaw); err != nil {
		return domain.User{}, err
	}
	user.Status = domain.UserStatus(statusRaw)
	if approvedByRaw.Valid {
		user.ApprovedBy = approvedByRaw.String
	}
	if approvedAtRaw.Valid && approvedAtRaw.String != "" {
		t, err := time.Parse(time.RFC3339Nano, approvedAtRaw.String)
		if err != nil {
			return domain.User{}, err
		}
		user.ApprovedAt = &t
	}
	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return domain.User{}, err
	}
	user.CreatedAt = createdAt
	return user, nil
}
