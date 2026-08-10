package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var ErrPasswordTooShort = errors.New("password must be at least 8 characters")
var ErrPasswordTooLong = errors.New("password must be at most 72 characters")

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}
	if len(password) > 72 {
		return ErrPasswordTooLong
	}
	return nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
