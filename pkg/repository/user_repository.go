package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/olezhek28/auth-service/pkg/models"
)

// UserRepository интерфейс для работы с пользователями
type UserRepository interface {
	CreateUser(ctx context.Context, req models.CreateUserRequest) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByUUID(ctx context.Context, userUUID uuid.UUID) (*models.User, error)
}

// userRepository реализация репозитория пользователей
type userRepository struct {
	db *pgxpool.Pool
	qb squirrel.StatementBuilderType
}

// NewUserRepository создает новый репозиторий пользователей
func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepository{
		db: db,
		qb: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// CreateUser создает нового пользователя
func (r *userRepository) CreateUser(ctx context.Context, req models.CreateUserRequest) (*models.User, error) {
	// Хешируем пароль
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	userUUID := uuid.New()
	now := time.Now()

	// Строим SQL запрос
	query, args, err := r.qb.
		Insert("users").
		Columns("uuid", "email", "username", "password_hash", "created_at", "updated_at").
		Values(userUUID, req.Email, req.Username, string(passwordHash), now, now).
		Suffix("RETURNING id, uuid, email, username, password_hash, created_at, updated_at").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build insert query: %w", err)
	}

	// Выполняем запрос
	var user models.User
	err = r.db.QueryRow(ctx, query, args...).Scan(
		&user.ID,
		&user.UUID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

// GetUserByEmail получает пользователя по email
func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query, args, err := r.qb.
		Select("id", "uuid", "email", "username", "password_hash", "created_at", "updated_at").
		From("users").
		Where(squirrel.Eq{"email": email}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var user models.User
	err = r.db.QueryRow(ctx, query, args...).Scan(
		&user.ID,
		&user.UUID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetUserByUUID получает пользователя по UUID
func (r *userRepository) GetUserByUUID(ctx context.Context, userUUID uuid.UUID) (*models.User, error) {
	query, args, err := r.qb.
		Select("id", "uuid", "email", "username", "password_hash", "created_at", "updated_at").
		From("users").
		Where(squirrel.Eq{"uuid": userUUID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var user models.User
	err = r.db.QueryRow(ctx, query, args...).Scan(
		&user.ID,
		&user.UUID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}
