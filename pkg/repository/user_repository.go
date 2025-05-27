package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/olezhek28/auth-service/pkg/errors"
	"github.com/olezhek28/auth-service/pkg/models"
)

// UserRepository интерфейс для работы с пользователями
type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) error
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
func (r *userRepository) CreateUser(ctx context.Context, user *models.User) error {
	// Строим SQL запрос
	query, args, err := r.qb.
		Insert("users").
		Columns("uuid", "email", "username", "password_hash", "created_at", "updated_at").
		Values(user.UUID, user.Email, user.Username, user.PasswordHash, user.CreatedAt, user.UpdatedAt).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	// Выполняем запрос
	err = r.db.QueryRow(ctx, query, args...).Scan(&user.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return apperrors.ErrUserAlreadyExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrUserNotFound
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}
