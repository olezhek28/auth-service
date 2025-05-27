package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	apperrors "github.com/olezhek28/auth-service/pkg/errors"
	"github.com/olezhek28/auth-service/pkg/logger"
	"github.com/olezhek28/auth-service/pkg/models"
	"github.com/olezhek28/auth-service/pkg/repository"
	"github.com/olezhek28/auth-service/pkg/validator"
)

// AuthService интерфейс сервиса аутентификации
type AuthService interface {
	Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error)
	Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
	WhoAmI(ctx context.Context, req WhoAmIRequest) (*WhoAmIResponse, error)
}

// RegisterRequest запрос на регистрацию
type RegisterRequest struct {
	Email    string
	Username string
	Password string
}

// RegisterResponse ответ на регистрацию
type RegisterResponse struct {
	UserUUID uuid.UUID
}

// LoginRequest запрос на вход
type LoginRequest struct {
	Email    string
	Password string
}

// LoginResponse ответ на вход
type LoginResponse struct {
	SessionUUID string
}

// WhoAmIRequest запрос информации о пользователе
type WhoAmIRequest struct {
	SessionUUID string
}

// WhoAmIResponse ответ с информацией о пользователе
type WhoAmIResponse struct {
	UserUUID  uuid.UUID
	Email     string
	Username  string
	CreatedAt time.Time
}

// authService реализация сервиса аутентификации
type authService struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	logger      logger.Logger
	sessionTTL  time.Duration
}

// NewAuthService создает новый сервис аутентификации
func NewAuthService(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	logger logger.Logger,
	sessionTTL time.Duration,
) AuthService {
	return &authService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		logger:      logger,
		sessionTTL:  sessionTTL,
	}
}

// Register регистрирует нового пользователя
func (s *authService) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	// Валидация входных данных
	if err := validator.ValidateEmail(req.Email); err != nil {
		return nil, err
	}
	if err := validator.ValidateUsername(req.Username); err != nil {
		return nil, err
	}
	if err := validator.ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	// Проверяем, что пользователь не существует
	existingUser, err := s.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, apperrors.ErrUserNotFound) {
		s.logger.Error("failed to check existing user", "error", err, "email", req.Email)
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, apperrors.ErrUserAlreadyExists
	}

	// Хешируем пароль
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to hash password", "error", err)
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Создаем пользователя
	user := &models.User{
		UUID:         uuid.New(),
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: string(passwordHash),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		s.logger.Error("failed to create user", "error", err, "email", req.Email)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Info("user registered successfully", "user_uuid", user.UUID, "email", req.Email)

	return &RegisterResponse{
		UserUUID: user.UUID,
	}, nil
}

// Login выполняет вход пользователя в систему
func (s *authService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	// Валидация входных данных
	if err := validator.ValidateEmail(req.Email); err != nil {
		return nil, err
	}
	if err := validator.ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	// Получаем пользователя по email
	user, err := s.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			return nil, apperrors.ErrInvalidCredentials
		}
		s.logger.Error("failed to get user", "error", err, "email", req.Email)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Проверяем пароль
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	// Создаем сессию
	sessionUUID, err := s.sessionRepo.CreateSession(ctx, user.UUID, s.sessionTTL)
	if err != nil {
		s.logger.Error("failed to create session", "error", err, "user_uuid", user.UUID)
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	s.logger.Info("user logged in successfully", "user_uuid", user.UUID, "session_uuid", sessionUUID)

	return &LoginResponse{
		SessionUUID: sessionUUID,
	}, nil
}

// WhoAmI возвращает информацию о текущем пользователе
func (s *authService) WhoAmI(ctx context.Context, req WhoAmIRequest) (*WhoAmIResponse, error) {
	// Валидация входных данных
	if err := validator.ValidateSessionUUID(req.SessionUUID); err != nil {
		return nil, err
	}

	// Получаем UUID пользователя из сессии
	userUUID, err := s.sessionRepo.GetSession(ctx, req.SessionUUID)
	if err != nil {
		if errors.Is(err, apperrors.ErrSessionNotFound) {
			return nil, apperrors.ErrSessionNotFound
		}
		s.logger.Error("failed to get session", "error", err, "session_uuid", req.SessionUUID)
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Получаем пользователя
	user, err := s.userRepo.GetUserByUUID(ctx, userUUID)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			// Удаляем невалидную сессию
			_ = s.sessionRepo.DeleteSession(ctx, req.SessionUUID)
			return nil, apperrors.ErrSessionNotFound
		}
		s.logger.Error("failed to get user", "error", err, "user_uuid", userUUID)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &WhoAmIResponse{
		UserUUID:  user.UUID,
		Email:     user.Email,
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
	}, nil
}
