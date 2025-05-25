package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	auth_v1 "github.com/olezhek28/auth-service/pkg/auth/v1"
	"github.com/olezhek28/auth-service/pkg/database"
	"github.com/olezhek28/auth-service/pkg/migrations"
	"github.com/olezhek28/auth-service/pkg/models"
	"github.com/olezhek28/auth-service/pkg/redis"
	"github.com/olezhek28/auth-service/pkg/repository"
)

const grpcPort = ":50051"

// AuthServer реализует AuthService
type AuthServer struct {
	auth_v1.UnimplementedAuthServiceServer
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
}

// NewAuthServer создает новый экземпляр AuthServer
func NewAuthServer(userRepo repository.UserRepository, sessionRepo repository.SessionRepository) *AuthServer {
	return &AuthServer{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

// Login реализует метод входа в систему
func (s *AuthServer) Login(ctx context.Context, req *auth_v1.LoginRequest) (*auth_v1.LoginResponse, error) {
	log.Printf("Login request for email: %s", req.Email)

	// Валидация входных данных
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	// Получаем пользователя по email
	user, err := s.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		log.Printf("User not found for email %s: %v", req.Email, err)
		return nil, status.Error(codes.NotFound, "user not found")
	}

	// Проверяем пароль
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		log.Printf("Invalid password for user %s", req.Email)
		return nil, status.Error(codes.Unauthenticated, "invalid password")
	}

	// Создаем сессию в Redis
	sessionUUID, err := s.sessionRepo.CreateSession(user.UUID)
	if err != nil {
		log.Printf("Failed to create session for user %s: %v", req.Email, err)
		return nil, status.Error(codes.Internal, "failed to create session")
	}

	log.Printf("User %s logged in successfully, session: %s", req.Email, sessionUUID)

	return &auth_v1.LoginResponse{
		SessionUuid: sessionUUID,
	}, nil
}

// Register реализует метод регистрации пользователя
func (s *AuthServer) Register(ctx context.Context, req *auth_v1.RegisterRequest) (*auth_v1.RegisterResponse, error) {
	log.Printf("Register request for email: %s, username: %s", req.Email, req.Username)

	// Валидация входных данных
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	if req.Username == "" {
		return nil, status.Error(codes.InvalidArgument, "username is required")
	}
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}
	if len(req.Password) < 6 {
		return nil, status.Error(codes.InvalidArgument, "password must be at least 6 characters")
	}

	// Проверяем, не существует ли уже пользователь с таким email
	existingUser, err := s.userRepo.GetUserByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, status.Error(codes.AlreadyExists, "user with this email already exists")
	}

	// Создаем пользователя
	createReq := models.CreateUserRequest{
		Email:    req.Email,
		Username: req.Username,
		Password: req.Password,
	}

	user, err := s.userRepo.CreateUser(ctx, createReq)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	log.Printf("User created successfully: ID=%d, UUID=%s", user.ID, user.UUID.String())

	return &auth_v1.RegisterResponse{
		UserUuid: user.ID,
	}, nil
}

// WhoAmI реализует метод получения информации о пользователе
func (s *AuthServer) WhoAmI(ctx context.Context, req *auth_v1.WhoAmIRequest) (*auth_v1.WhoAmIResponse, error) {
	log.Printf("WhoAmI request for session: %s", req.SessionUuid)

	// Валидация входных данных
	if req.SessionUuid == "" {
		return nil, status.Error(codes.InvalidArgument, "session_uuid is required")
	}

	// Получаем UUID пользователя из сессии
	userUUID, err := s.sessionRepo.GetSession(req.SessionUuid)
	if err != nil {
		log.Printf("Session not found: %s, error: %v", req.SessionUuid, err)
		return nil, status.Error(codes.Unauthenticated, "invalid session")
	}

	// Получаем пользователя по UUID
	user, err := s.userRepo.GetUserByUUID(ctx, userUUID)
	if err != nil {
		log.Printf("User not found for UUID %s: %v", userUUID.String(), err)
		return nil, status.Error(codes.NotFound, "user not found")
	}

	log.Printf("WhoAmI successful for user: %s (UUID: %s)", user.Email, user.UUID.String())

	return &auth_v1.WhoAmIResponse{
		UserUuid:  user.ID,
		Email:     user.Email,
		Username:  user.Username,
		CreatedAt: timestamppb.New(user.CreatedAt),
	}, nil
}

func main() {
	ctx := context.Background()

	// Настройки БД (в реальном проекте из переменных окружения)
	dbConfig := database.Config{
		Host:     "localhost",
		Port:     "5432",
		Database: "auth_db",
		Username: "auth_user",
		Password: "auth_password",
		SSLMode:  "disable",
	}

	// Настройки Redis
	redisConfig := redis.Config{
		Host:     "localhost",
		Port:     "6379",
		Password: "",
		DB:       0,
	}

	// Формируем DSN для миграций
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbConfig.Username,
		dbConfig.Password,
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.Database,
		dbConfig.SSLMode,
	)

	// Применяем миграции
	log.Println("🔄 Applying database migrations...")
	if err := migrations.RunMigrations(dsn); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("✅ Database migrations applied successfully")

	// Подключаемся к БД
	pool, err := database.NewPostgresPool(ctx, dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	log.Println("✅ Connected to PostgreSQL")

	// Подключаемся к Redis
	redisPool := redis.NewRedisPool(redisConfig)
	defer redisPool.Close()

	// Проверяем соединение с Redis
	conn := redisPool.Get()
	_, err = conn.Do("PING")
	conn.Close()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("✅ Connected to Redis")

	// Создаем репозитории
	userRepo := repository.NewUserRepository(pool)
	sessionRepo := repository.NewSessionRepository(redisPool)

	// Создаем TCP listener
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", grpcPort, err)
	}

	// Создаем gRPC сервер
	grpcServer := grpc.NewServer()

	// Регистрируем наш сервис
	authServer := NewAuthServer(userRepo, sessionRepo)
	auth_v1.RegisterAuthServiceServer(grpcServer, authServer)

	// Включаем reflection для удобства отладки
	reflection.Register(grpcServer)

	// Канал для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем сервер в горутине
	go func() {
		log.Printf("🚀 gRPC server starting on port %s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	// Ждем сигнал завершения
	<-quit
	log.Println("🛑 Shutting down gRPC server...")

	// Graceful shutdown с таймаутом
	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		log.Println("✅ gRPC server stopped gracefully")
	case <-time.After(30 * time.Second):
		log.Println("⚠️ Force stopping gRPC server")
		grpcServer.Stop()
	}
}
