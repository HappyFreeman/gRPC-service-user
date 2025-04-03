package service

import (
	"context"
	"github.com/HappyFreeman/gRPC-service-user/internal/config"
	pb "github.com/HappyFreeman/gRPC-service-user/internal/proto/gen"
	"github.com/HappyFreeman/gRPC-service-user/internal/repo"
	"github.com/HappyFreeman/gRPC-service-user/pkg/jwt"
	"github.com/HappyFreeman/gRPC-service-user/pkg/pass"
	"github.com/HappyFreeman/gRPC-service-user/pkg/validator"
	"go.uber.org/zap"
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
	repo   repo.Repository
	log    *zap.SugaredLogger
	cfgJWT config.JWT
	pb.UnimplementedAuthServiceServer
}

func NewService(repo repo.Repository, logger *zap.SugaredLogger, cfgJWT config.JWT) Service {
	return &service{
		repo:   repo,
		log:    logger,
		cfgJWT: cfgJWT,
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
	hash, err := pass.HashPassword(request.GetPassword())

	if err != nil {
		s.log.Errorf("failed to hash password: %s", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	id, err := s.repo.CreateUser(ctx, repo.User{
		Name:     request.GetUsername(),
		Password: hash,
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
	if err := pass.CheckPassword(request.GetPassword(), user.Password); err != nil {
		return nil, status.Error(codes.NotFound, "invalid username or password")
	}

	// Создание токена
	token, err := jwt.NewToken(user, s.cfgJWT)

	if err != nil {
		s.log.Errorf("failed to create token: %s", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.LoginResponse{Token: token}, nil

}

func (s *service) ChangePassword(ctx context.Context, request *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	if err := validator.Validate(ctx, request); err != nil {
		s.log.Errorf("validation error: %s", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	user, err := s.repo.GetUserByName(ctx, request.GetUsername())
	if err != nil {
		return nil, status.Error(codes.NotFound, "invalid username or password")
	}

	if err := pass.CheckPassword(request.GetOldPassword(), user.Password); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid argument")
	}

	hash, err := pass.HashPassword(request.GetNewPassword())

	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	user.Password = hash

	if err := s.repo.ChangePassword(ctx, user); err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.ChangePasswordResponse{Message: "Password changed successfully"}, nil
}

func (s *service) ValidateToken(ctx context.Context, request *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	if err := validator.Validate(ctx, request); err != nil {
		s.log.Errorf("validation error: %s", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	id, err := jwt.GetUserId(request.GetToken(), s.cfgJWT.Secret)

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &pb.ValidateTokenResponse{Message: strconv.Itoa(id)}, nil
}
