package service

import (
	"context"
	"database/sql"
	pb "github.com/HappyFreeman/gRPC-service-user/grpc/genproto"
	"github.com/HappyFreeman/gRPC-service-user/internal/config"
	"github.com/HappyFreeman/gRPC-service-user/internal/repo"
	"github.com/HappyFreeman/gRPC-service-user/pkg/jwt"
	"github.com/HappyFreeman/gRPC-service-user/pkg/pass"
	"github.com/HappyFreeman/gRPC-service-user/pkg/validator"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strconv"
	"time"
)

// Слой бизнес-логики. Тут должна быть основная логика сервиса

// Service - интерфейс для бизнес-логики
type Service interface {
	pb.AuthServiceServer
}
type service struct {
	cfg                   config.AppConfig
	repo                  repo.Repository
	log                   *zap.SugaredLogger
	jwt                   jwt.JWTClient
	numberPasswordEntries *cache.Cache
	pb.UnimplementedAuthServiceServer
}

func NewService(cfg config.AppConfig, repo repo.Repository, logger *zap.SugaredLogger, jwt jwt.JWTClient) Service {
	return &service{
		cfg:  cfg,
		repo: repo,
		log:  logger,
		jwt:  jwt,
		numberPasswordEntries: cache.New(
			cfg.System.LockPasswordEntry,
			cfg.System.LockPasswordEntry,
		),
	}
}

func (s *service) Register(ctx context.Context, request *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// Валидация входных данных
	if err := validator.Validate(ctx, request); err != nil {
		s.log.Errorf("validation error: %s", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	passwordValidityCheck, err := pass.IsValidPassword(request.GetPassword())

	if !passwordValidityCheck {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Проверка существования пользователя
	if _, exists := s.repo.GetUserByName(ctx, request.GetUsername()); exists == nil {
		return nil, status.Errorf(codes.AlreadyExists, ErrUserAuthAlreadyExist)
	}

	// Хеширование пароля
	hash, err := pass.HashPassword(request.GetPassword())

	if err != nil {
		s.log.Errorf("failed to hash password: %s", err)
		return nil, status.Error(codes.Internal, ErrUnknown)
	}

	id, err := s.repo.CreateUser(ctx, repo.User{
		Name:     request.GetUsername(),
		Password: hash,
	})

	if err != nil {
		s.log.Errorf("failed to create user: %s", err)
		return nil, status.Error(codes.Internal, ErrUnknown)
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
		return nil, status.Error(codes.NotFound, ErrUserNotFound)
	}

	// Проверка пароля
	if err := pass.CheckPassword(request.GetPassword(), user.Password); err != nil {
		return nil, status.Error(codes.NotFound, ErrValidatePassword)
	}

	// Создание токена
	tokens, err := s.jwt.CreateToken(&jwt.CreateTokenParams{
		UserId: user.ID,
	})

	if err != nil {
		s.log.Errorf("failed to create token: %s", err)
		return nil, status.Error(codes.Internal, ErrUnknown)
	}

	_, err = s.repo.NewRefreshToken(ctx, repo.NewRefreshTokenParams{
		UserID:       user.ID,
		Token:        tokens.RefreshToken,
		ExpirationAt: sql.NullTime{Time: time.Now().Add(s.cfg.System.RefreshTokenTimeout), Valid: true},
	})

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.ForeignKeyViolation {
				return nil, status.Error(codes.NotFound, ErrUserNotFound)
			}
		}
		s.log.Errorf("adding a token to the database: user_id = %d", user.ID)
		return nil, status.Error(codes.Internal, ErrUnknown)
	}

	return &pb.LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil

}

func (s *service) ChangePassword(ctx context.Context, request *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {

	remainingAttempts, err := s.checkRemainingAttempts(request.GetUsername())
	if err != nil {
		return nil, err
	}

	passwordValidityCheck, err := pass.IsValidPassword(request.NewPassword)

	if !passwordValidityCheck {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	user, err := s.repo.GetUserByName(ctx, request.GetUsername())
	if err != nil {
		return nil, status.Error(codes.NotFound, ErrUserNotFound)
	}

	if err := pass.CheckPassword(request.GetOldPassword(), user.Password); err != nil {
		s.numberPasswordEntries.Set(request.GetUsername(), remainingAttempts-1, cache.DefaultExpiration)
		return nil, status.Errorf(
			codes.InvalidArgument,
			"%s %d",
			ErrValidatePassword,
			remainingAttempts-1,
		)
	}

	// новый пароль не должен совпадать со старым
	err = pass.CheckPassword(request.GetOldPassword(), user.Password)
	if err == nil {
		return nil, status.Error(codes.InvalidArgument, ErrPasswordMatchOldPassword)
	}

	hash, err := pass.HashPassword(request.GetNewPassword())

	if err != nil {
		return nil, status.Error(codes.Internal, ErrUnknown)
	}

	user.Password = hash

	err = s.repo.ChangePassword(ctx, repo.UpdatePasswordParams{
		Password: hash,
		UserID:   user.ID,
	})

	if err != nil {
		return nil, status.Error(codes.Internal, ErrUnknown)
	}

	s.numberPasswordEntries.Delete(request.GetUsername())

	return &pb.ChangePasswordResponse{Message: "Password changed successfully"}, nil
}

func (s *service) ValidateToken(ctx context.Context, request *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	if err := validator.Validate(ctx, request); err != nil {
		s.log.Errorf("validation error: %s", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	check, err := s.jwt.ValidateToken(&jwt.ValidateTokenParams{
		Token: request.Token,
	})

	if err != nil || !check {
		return nil, status.Error(codes.Unauthenticated, ErrValidateJwt)
	}

	accessData, err := s.jwt.GetDataFromToken(&jwt.GetDataFromTokenParams{
		Token: request.Token,
	})

	if err != nil {
		return nil, status.Error(codes.Unauthenticated, ErrValidateJwt)
	}

	_, err = s.repo.GetRefreshToken(ctx, repo.GetRefreshTokenParams{
		UserID: accessData.UserId,
	})

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.Unauthenticated, ErrValidateJwt)
		}

		return nil, status.Error(codes.Internal, ErrUnknown)
	}

	return &pb.ValidateTokenResponse{Message: strconv.Itoa(int(accessData.UserId))}, nil
}

func (s *service) Refresh(ctx context.Context, request *pb.RefreshRequest) (*pb.RefreshResponse, error) {

	check, err := s.jwt.ValidateToken(&jwt.ValidateTokenParams{
		Token: request.RefreshToken,
	})
	if err != nil || !check {
		s.log.Errorf("validate refresh token err")
		return nil, status.Error(codes.Unauthenticated, ErrValidateJwt)
	}

	accessData, err := s.jwt.GetDataFromToken(&jwt.GetDataFromTokenParams{
		Token: request.AccessToken,
	})

	if err != nil {
		return nil, status.Error(codes.Unauthenticated, ErrValidateJwt)
	}

	refreshData, err := s.jwt.GetDataFromToken(&jwt.GetDataFromTokenParams{
		Token: request.RefreshToken,
	})
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, ErrValidateJwt)
	}
	if accessData.UserId != refreshData.UserId {
		return nil, status.Error(codes.Unauthenticated, ErrValidateJwt)
	}

	rtToken, err := s.repo.GetRefreshToken(ctx, repo.GetRefreshTokenParams{
		UserID: refreshData.UserId,
	})
	if err != nil {
		s.log.Errorf("get refresh token err")
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, ErrTokenNotFound)
		}
		return nil, status.Error(codes.Internal, ErrUnknown)
	}

	if len(rtToken) == 0 {
		s.log.Errorf("len(rtToken) == 0")
		return nil, status.Error(codes.NotFound, ErrTokenNotFound)
	}

	if rtToken[0] != request.RefreshToken {
		s.log.Errorf("rtToken[0] != req.RefreshToken")
		return nil, status.Error(codes.Unauthenticated, ErrValidateJwt)
	}

	tokens, err := s.jwt.CreateToken(&jwt.CreateTokenParams{
		UserId: refreshData.UserId,
	})

	if err != nil {
		s.log.Errorf("create tokens error")
		return nil, status.Error(codes.Internal, ErrUnknown)
	}

	err = s.repo.UpdateRefreshToken(ctx, repo.UpdateRefreshTokenParams{
		Token:        tokens.RefreshToken,
		ExpirationAt: sql.NullTime{Time: time.Now().Add(s.cfg.System.RefreshTokenTimeout), Valid: true},
		UserID:       refreshData.UserId,
	})

	if err != nil {
		s.log.Errorf("update refresh token err")
		return nil, status.Error(codes.Internal, ErrUnknown)
	}

	return &pb.RefreshResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (s *service) checkRemainingAttempts(userName string) (int64, error) {

	remainingAttempts := s.cfg.System.NumberPasswordAttempts
	remainingAttemptsFromCache, expirationTime, ok := s.numberPasswordEntries.GetWithExpiration(userName)

	if ok && remainingAttemptsFromCache.(int64) == 0 {
		return 0, lockForActionErr(expirationTime)
	}
	if ok {
		remainingAttempts = remainingAttemptsFromCache.(int64)
	}
	return remainingAttempts, nil

}
