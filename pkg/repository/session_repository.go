package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"

	apperrors "github.com/olezhek28/auth-service/pkg/errors"
)

// SessionRepository интерфейс для работы с сессиями
type SessionRepository interface {
	CreateSession(ctx context.Context, userUUID uuid.UUID, ttl time.Duration) (string, error)
	GetSession(ctx context.Context, sessionUUID string) (uuid.UUID, error)
	DeleteSession(ctx context.Context, sessionUUID string) error
}

// sessionRepository реализация репозитория сессий
type sessionRepository struct {
	pool *redis.Pool
}

// NewSessionRepository создает новый репозиторий сессий
func NewSessionRepository(pool *redis.Pool) SessionRepository {
	return &sessionRepository{
		pool: pool,
	}
}

// CreateSession создает новую сессию для пользователя
func (r *sessionRepository) CreateSession(ctx context.Context, userUUID uuid.UUID, ttl time.Duration) (string, error) {
	conn := r.pool.Get()
	defer conn.Close()

	sessionUUID := uuid.New().String()
	sessionKey := fmt.Sprintf("session:%s", sessionUUID)

	// Сохраняем сессию с указанным TTL
	_, err := conn.Do("SETEX", sessionKey, int(ttl.Seconds()), userUUID.String())
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return sessionUUID, nil
}

// GetSession получает UUID пользователя по UUID сессии
func (r *sessionRepository) GetSession(ctx context.Context, sessionUUID string) (uuid.UUID, error) {
	conn := r.pool.Get()
	defer conn.Close()

	sessionKey := fmt.Sprintf("session:%s", sessionUUID)

	userUUIDStr, err := redis.String(conn.Do("GET", sessionKey))
	if err != nil {
		if err == redis.ErrNil {
			return uuid.Nil, apperrors.ErrSessionNotFound
		}
		return uuid.Nil, fmt.Errorf("failed to get session: %w", err)
	}

	userUUID, err := uuid.Parse(userUUIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user UUID in session: %w", err)
	}

	return userUUID, nil
}

// DeleteSession удаляет сессию
func (r *sessionRepository) DeleteSession(ctx context.Context, sessionUUID string) error {
	conn := r.pool.Get()
	defer conn.Close()

	sessionKey := fmt.Sprintf("session:%s", sessionUUID)

	_, err := conn.Do("DEL", sessionKey)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}
