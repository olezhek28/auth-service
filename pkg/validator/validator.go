package validator

import (
	"fmt"
	"net/mail"
	"strings"

	apperrors "github.com/olezhek28/auth-service/pkg/errors"
)

// ValidateEmail проверяет корректность email
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("%w: email is required", apperrors.ErrInvalidInput)
	}

	email = strings.TrimSpace(email)
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("%w: invalid email format", apperrors.ErrInvalidInput)
	}

	return nil
}

// ValidatePassword проверяет корректность пароля
func ValidatePassword(password string) error {
	if password == "" {
		return fmt.Errorf("%w: password is required", apperrors.ErrInvalidInput)
	}

	if len(password) < 6 {
		return fmt.Errorf("%w: password must be at least 6 characters", apperrors.ErrInvalidInput)
	}

	return nil
}

// ValidateUsername проверяет корректность имени пользователя
func ValidateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("%w: username is required", apperrors.ErrInvalidInput)
	}

	username = strings.TrimSpace(username)
	if len(username) < 2 {
		return fmt.Errorf("%w: username must be at least 2 characters", apperrors.ErrInvalidInput)
	}

	if len(username) > 50 {
		return fmt.Errorf("%w: username must be less than 50 characters", apperrors.ErrInvalidInput)
	}

	return nil
}

// ValidateSessionUUID проверяет корректность UUID сессии
func ValidateSessionUUID(sessionUUID string) error {
	if sessionUUID == "" {
		return fmt.Errorf("%w: session_uuid is required", apperrors.ErrInvalidInput)
	}

	sessionUUID = strings.TrimSpace(sessionUUID)
	if len(sessionUUID) != 36 {
		return fmt.Errorf("%w: invalid session_uuid format", apperrors.ErrInvalidInput)
	}

	return nil
}
