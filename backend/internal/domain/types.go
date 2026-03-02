package domain

import "time"

// User is the core account model in our system.
type User struct {
	ID         string     `json:"id"`
	Email      string     `json:"email"`
	Status     UserStatus `json:"status"`
	ApprovedBy string     `json:"approvedBy,omitempty"`
	ApprovedAt *time.Time `json:"approvedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

type UserStatus string

const (
	UserStatusPending  UserStatus = "pending"
	UserStatusApproved UserStatus = "approved"
	UserStatusRejected UserStatus = "rejected"
)

type Provider string

const (
	// ProviderSpotify represents Spotify integration.
	ProviderSpotify Provider = "spotify"
	// ProviderYouTubeMusic represents YouTube Music integration.
	ProviderYouTubeMusic = "youtube_music"
)

// ConnectedAccount stores provider credentials and sync metadata for one user+provider pair.
// Access/refresh tokens are stored encrypted.
type ConnectedAccount struct {
	UserID              string    `json:"userId"`
	Provider            Provider  `json:"provider"`
	EncryptedAccess     string    `json:"-"`
	EncryptedRefresh    string    `json:"-"`
	TokenExpiryUnix     int64     `json:"tokenExpiryUnix"`
	LastSyncCheckpoint  string    `json:"lastSyncCheckpoint"`
	ConnectedAt         time.Time `json:"connectedAt"`
	LastRotationUnixSec int64     `json:"lastRotationUnixSec"`
}

type SyncJobStatus string

const (
	// Sync job lifecycle: queued -> running -> complete/failed.
	SyncJobPending  SyncJobStatus = "pending"
	SyncJobRunning                = "running"
	SyncJobComplete               = "complete"
	SyncJobFailed                 = "failed"
)

// SyncJob tracks one playlist sync request from a source provider to destination provider.
type SyncJob struct {
	ID          string        `json:"id"`
	UserID      string        `json:"userId"`
	Source      Provider      `json:"source"`
	Destination Provider      `json:"destination"`
	PlaylistID  string        `json:"playlistId"`
	Status      SyncJobStatus `json:"status"`
	Error       string        `json:"error,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
}
