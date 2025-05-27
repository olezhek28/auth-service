package errors

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Предопределенные ошибки
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrSessionNotFound    = errors.New("session not found")
	ErrInvalidInput       = errors.New("invalid input")
	ErrInternal           = errors.New("internal error")
)

// AppError представляет ошибку приложения с дополнительным контекстом
type AppError struct {
	Code    codes.Code
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// ToGRPCError конвертирует ошибку в gRPC статус
func (e *AppError) ToGRPCError() error {
	return status.Error(e.Code, e.Message)
}

// New создает новую ошибку приложения
func New(code codes.Code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap оборачивает ошибку с дополнительным контекстом
func Wrap(err error, code codes.Code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// FromError конвертирует стандартную ошибку в AppError
func FromError(err error) *AppError {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	// Маппинг известных ошибок
	switch {
	case errors.Is(err, ErrUserNotFound):
		return New(codes.NotFound, "User not found")
	case errors.Is(err, ErrUserAlreadyExists):
		return New(codes.AlreadyExists, "User already exists")
	case errors.Is(err, ErrInvalidCredentials):
		return New(codes.Unauthenticated, "Invalid credentials")
	case errors.Is(err, ErrSessionNotFound):
		return New(codes.Unauthenticated, "Session not found")
	case errors.Is(err, ErrInvalidInput):
		return New(codes.InvalidArgument, "Invalid input")
	default:
		return New(codes.Internal, "Internal server error")
	}
}
