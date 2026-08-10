package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
)

const SessionTTL = 7 * 24 * time.Hour

var ErrSessionNotFound = errors.New("session not found or expired")

type Session struct {
	AdminID   int64
	ExpiresAt time.Time
}

type SessionStore struct {
	db *sql.DB
}

func NewSessionStore(db *sql.DB) *SessionStore {
	return &SessionStore{db: db}
}

func (s *SessionStore) Create(adminID int64) (token string, expiresAt time.Time, err error) {
	token, err = generateToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt = time.Now().Add(SessionTTL)
	_, err = s.db.Exec(
		`INSERT INTO sessions (admin_id, token_hash, expires_at) VALUES (?, ?, ?);`,
		adminID, hashToken(token), expiresAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

func (s *SessionStore) Validate(token string) (Session, error) {
	var adminID int64
	var expiresAtStr string
	err := s.db.QueryRow(
		`SELECT admin_id, expires_at FROM sessions WHERE token_hash = ?;`, hashToken(token),
	).Scan(&adminID, &expiresAtStr)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrSessionNotFound
	}
	if err != nil {
		return Session{}, err
	}

	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil {
		return Session{}, err
	}
	if time.Now().After(expiresAt) {
		return Session{}, ErrSessionNotFound
	}
	return Session{AdminID: adminID, ExpiresAt: expiresAt}, nil
}

func (s *SessionStore) Delete(token string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token_hash = ?;`, hashToken(token))
	return err
}

func (s *SessionStore) DeleteAllForAdmin(adminID int64) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE admin_id = ?;`, adminID)
	return err
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
