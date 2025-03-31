package service

import (
	"context"
	pb "github.com/HappyFreeman/gRPC-service-user/internal/proto/gen"
	"github.com/HappyFreeman/gRPC-service-user/internal/repo"
	"github.com/HappyFreeman/gRPC-service-user/pkg/validator"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strconv"
)

// Слой бизнес-логики. Тут должна быть основная логика сервиса

// Service - интерфейс для бизнес-логики
type Service interface {
	pb.AuthServiceServer
}
type service struct {
	repo repo.Repository
	log  *zap.SugaredLogger
	pb.UnimplementedAuthServiceServer
}

func NewService(repo repo.Repository, logger *zap.SugaredLogger) Service {
	return &service{
		repo: repo,
		log:  logger,
	}
}

func (s *service) Register(ctx context.Context, request *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// Валидация входных данных
	if err := validator.Validate(ctx, request); err != nil {
		s.log.Errorf("validation error: %s", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Проверка существования пользователя
	if _, exists := s.repo.GetUserByName(ctx, request.GetUsername()); exists == nil {
		return nil, status.Errorf(codes.AlreadyExists, "user already exists")
	}

	// Хеширование пароля
	hash, err := bcrypt.GenerateFromPassword([]byte(request.GetPassword()), bcrypt.DefaultCost)

	if err != nil {
		s.log.Errorf("failed to hash password: %s", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	id, err := s.repo.CreateUser(ctx, repo.User{
		Name:     request.GetUsername(),
		Password: string(hash),
	})

	if err != nil {
		s.log.Errorf("failed to create user: %s", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.RegisterResponse{Message: "User created successfully. id: " + strconv.Itoa(id)}, nil
}

func (s *service) Login(ctx context.Context, request *pb.LoginRequest) (*pb.LoginResponse, error) {
	// Валидация входных данных
	if err := validator.Validate(ctx, request); err != nil {
		s.log.Errorf("validation error: %s", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Проверка существования пользователя
	user, err := s.repo.GetUserByName(ctx, request.GetUsername())
	if err != nil {
		return nil, status.Error(codes.NotFound, "invalid username or password")
	}

	// Проверка пароля
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.GetPassword())); err != nil {
		return nil, status.Error(codes.NotFound, "invalid username or password")
	}

	return &pb.LoginResponse{Token: "token"}, nil

}
