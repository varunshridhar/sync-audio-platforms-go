package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type SessionClaims struct {
	UserID string `json:"uid"`
	Exp    int64  `json:"exp"`
}

type SessionManager struct {
	secret []byte
}

// NewSessionManager creates a signer/verifier for lightweight session tokens.
func NewSessionManager(secret string) *SessionManager {
	return &SessionManager{secret: []byte(secret)}
}

// Sign creates a compact token:
// base64url(payload JSON) + "." + base64url(HMAC signature).
func (m *SessionManager) Sign(userID string, ttl time.Duration) (string, error) {
	claims := SessionClaims{
		UserID: userID,
		Exp:    time.Now().Add(ttl).Unix(),
	}
	body, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	msg := base64.RawURLEncoding.EncodeToString(body)
	sig := m.sign(msg)
	return msg + "." + sig, nil
}

// Verify checks token structure, signature integrity, and expiration timestamp.
func (m *SessionManager) Verify(token string) (SessionClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return SessionClaims{}, errors.New("invalid token format")
	}
	msg, sig := parts[0], parts[1]
	if !hmac.Equal([]byte(m.sign(msg)), []byte(sig)) {
		return SessionClaims{}, errors.New("invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(msg)
	if err != nil {
		return SessionClaims{}, err
	}
	var claims SessionClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return SessionClaims{}, err
	}
	if time.Now().Unix() > claims.Exp {
		return SessionClaims{}, errors.New("session expired")
	}
	return claims, nil
}

func (m *SessionManager) sign(msg string) string {
	h := hmac.New(sha256.New, m.secret)
	h.Write([]byte(msg))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

