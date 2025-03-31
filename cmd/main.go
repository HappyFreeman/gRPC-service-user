package main

import (
	"context"
	"github.com/HappyFreeman/gRPC-service-user/internal/config"
	"github.com/HappyFreeman/gRPC-service-user/internal/repo"
	"github.com/HappyFreeman/gRPC-service-user/internal/service"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/HappyFreeman/gRPC-service-user/internal/proto/gen"
	customLogger "github.com/HappyFreeman/gRPC-service-user/pkg/logger"
)

func main() {

	// Загружаем env в переменные окружения
	if err := godotenv.Load(".env"); err != nil {
		log.Fatal(errors.Wrap(err, "failed to load environment variables"))
	}

	// Загружаем конфигурацию из переменных окружения
	var cfg config.AppConfig
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatal(errors.Wrap(err, "failed to load configuration"))
	}

	// Инициализация логгера
	logger, err := customLogger.NewLogger(cfg.LogLevel)
	if err != nil {
		log.Fatal(errors.Wrap(err, "error initializing logger"))
	}

	// Подключение к PostgreSQL
	repository, err := repo.NewRepository(context.Background(), cfg.PostgreSQL)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to initialize repository"))
	}

	// Создание сервиса с бизнес-логикой
	serviceInstance := service.NewService(repository, logger)

	//подключение gRPC-сервера.
	server := grpc.NewServer()
	pb.RegisterAuthServiceServer(server, serviceInstance)

	lis, err := net.Listen("tcp", ":"+cfg.GRPC.ListenAddress)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	go func() {
		logger.Infof("starting gRPC server on %s", cfg.GRPC.ListenAddress)
		if err := server.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Ожидание системных сигналов для корректного завершения работы
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan

	logger.Info("Shutting down gracefully...")
	server.GracefulStop()
}
