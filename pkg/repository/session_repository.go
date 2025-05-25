package repository

import (
	"fmt"

	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
)

// SessionRepository интерфейс для работы с сессиями
type SessionRepository interface {
	CreateSession(userUUID uuid.UUID) (string, error)
	GetSession(sessionUUID string) (uuid.UUID, error)
	DeleteSession(sessionUUID string) error
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
func (r *sessionRepository) CreateSession(userUUID uuid.UUID) (string, error) {
	conn := r.pool.Get()
	defer conn.Close()

	sessionUUID := uuid.New().String()
	sessionKey := fmt.Sprintf("session:%s", sessionUUID)

	// Сохраняем сессию на 24 часа
	_, err := conn.Do("SETEX", sessionKey, 86400, userUUID.String())
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return sessionUUID, nil
}

// GetSession получает UUID пользователя по UUID сессии
func (r *sessionRepository) GetSession(sessionUUID string) (uuid.UUID, error) {
	conn := r.pool.Get()
	defer conn.Close()

	sessionKey := fmt.Sprintf("session:%s", sessionUUID)

	userUUIDStr, err := redis.String(conn.Do("GET", sessionKey))
	if err != nil {
		if err == redis.ErrNil {
			return uuid.Nil, fmt.Errorf("session not found")
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
func (r *sessionRepository) DeleteSession(sessionUUID string) error {
	conn := r.pool.Get()
	defer conn.Close()

	sessionKey := fmt.Sprintf("session:%s", sessionUUID)

	_, err := conn.Do("DEL", sessionKey)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}
