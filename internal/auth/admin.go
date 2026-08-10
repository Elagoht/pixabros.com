package auth

import (
	"database/sql"
	"errors"
)

var ErrAdminNotFound = errors.New("admin not found")

type Admin struct {
	ID           int64
	Username     string
	PasswordHash string
}

type AdminRepo struct {
	db *sql.DB
}

func NewAdminRepo(db *sql.DB) *AdminRepo {
	return &AdminRepo{db: db}
}

func (r *AdminRepo) FindByUsername(username string) (Admin, error) {
	var a Admin
	err := r.db.QueryRow(
		`SELECT id, username, password_hash FROM admins WHERE username = ?;`, username,
	).Scan(&a.ID, &a.Username, &a.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return Admin{}, ErrAdminNotFound
	}
	if err != nil {
		return Admin{}, err
	}
	return a, nil
}

func (r *AdminRepo) FindByID(id int64) (Admin, error) {
	var a Admin
	err := r.db.QueryRow(
		`SELECT id, username, password_hash FROM admins WHERE id = ?;`, id,
	).Scan(&a.ID, &a.Username, &a.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return Admin{}, ErrAdminNotFound
	}
	if err != nil {
		return Admin{}, err
	}
	return a, nil
}

func (r *AdminRepo) UpdatePasswordHash(id int64, hash string) error {
	_, err := r.db.Exec(`UPDATE admins SET password_hash = ? WHERE id = ?;`, hash, id)
	return err
}
