package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	auth_v1 "github.com/olezhek28/auth-service/pkg/auth/v1"
	"github.com/olezhek28/auth-service/pkg/config"
	"github.com/olezhek28/auth-service/pkg/database"
	"github.com/olezhek28/auth-service/pkg/handler"
	"github.com/olezhek28/auth-service/pkg/interceptor"
	"github.com/olezhek28/auth-service/pkg/logger"
	"github.com/olezhek28/auth-service/pkg/migrations"
	"github.com/olezhek28/auth-service/pkg/redis"
	"github.com/olezhek28/auth-service/pkg/repository"
	"github.com/olezhek28/auth-service/pkg/service"
)

func main() {
	ch := make(chan struct{}, 1)
	ch <- struct{}{}
	val, ok := <-ch
	fmt.Println(val, ok)
	close(ch)
	val, ok = <-ch
	fmt.Println(val, ok)

	return
	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Создаем логгер
	log := logger.NewDevelopment()
	log.Info("starting auth service", "config", cfg)

	// Создаем контекст с отменой
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Применяем миграции
	log.Info("applying database migrations")
	if err := migrations.RunMigrations(cfg.Database.DSN()); err != nil {
		log.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	log.Info("database migrations applied successfully")

	// Подключаемся к PostgreSQL
	dbPool, err := database.NewPostgresPool(ctx, database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		Database: cfg.Database.Database,
		Username: cfg.Database.Username,
		Password: cfg.Database.Password,
		SSLMode:  cfg.Database.SSLMode,
	})
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()
	log.Info("connected to PostgreSQL")

	// Подключаемся к Redis
	redisPool := redis.NewRedisPool(redis.Config{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisPool.Close()

	// Проверяем соединение с Redis
	conn := redisPool.Get()
	_, err = conn.Do("PING")
	conn.Close()
	if err != nil {
		log.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	log.Info("connected to Redis")

	// Создаем репозитории
	userRepo := repository.NewUserRepository(dbPool)
	sessionRepo := repository.NewSessionRepository(redisPool)

	// Создаем сервисы
	authService := service.NewAuthService(userRepo, sessionRepo, log, cfg.Auth.SessionTTL)

	// Создаем handlers
	authHandler := handler.NewAuthHandler(authService, log)

	// Создаем TCP listener
	lis, err := net.Listen("tcp", cfg.Server.Port)
	if err != nil {
		log.Error("failed to listen", "port", cfg.Server.Port, "error", err)
		os.Exit(1)
	}

	// Создаем gRPC сервер с interceptors
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptor.RecoveryInterceptor(log),
			interceptor.LoggingInterceptor(log),
		),
	)

	// Регистрируем сервисы
	auth_v1.RegisterAuthServiceServer(grpcServer, authHandler)

	// Включаем reflection для отладки
	reflection.Register(grpcServer)

	// Запускаем сервер в горутине
	go func() {
		log.Info("starting gRPC server", "port", cfg.Server.Port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Error("gRPC server failed", "error", err)
			cancel()
		}
	}()

	// Ожидаем сигнал завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Info("received shutdown signal", "signal", sig)
	case <-ctx.Done():
		log.Info("context cancelled")
	}

	// Graceful shutdown
	log.Info("shutting down server")

	// Создаем контекст с таймаутом для shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	// Останавливаем gRPC сервер
	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		log.Info("server stopped gracefully")
	case <-shutdownCtx.Done():
		log.Warn("shutdown timeout exceeded, forcing stop")
		grpcServer.Stop()
	}

	log.Info("auth service stopped")
}
