package suite

import (
	"context"
	pb "github.com/HappyFreeman/gRPC-service-user/grpc/genproto"
	"github.com/HappyFreeman/gRPC-service-user/internal/config"
	"github.com/HappyFreeman/gRPC-service-user/pkg/jwt"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net"
	"testing"
)

const (
	grpcHost = "localhost"
)

type Suite struct {
	*testing.T
	Cfg               *config.AppConfig
	JWT               jwt.JWTClient
	AuthServiceClient pb.AuthServiceClient
}

func New(t *testing.T) (context.Context, *Suite) {
	t.Helper()
	t.Parallel()

	// Загружаем env в переменные окружения
	if err := godotenv.Load("../.env"); err != nil {
		t.Fatal(errors.Wrap(err, "failed to load environment variables"))
	}

	// Загружаем конфигурацию из переменных окружения
	var cfg config.AppConfig
	if err := envconfig.Process("", &cfg); err != nil {
		t.Fatal(errors.Wrap(err, "failed to load configuration"))
	}

	ctx, cancelCtx := context.WithTimeout(context.Background(), cfg.PostgreSQL.PoolMaxConnIdleTime)

	t.Cleanup(func() {
		t.Helper()
		cancelCtx()
	})

	cc, err := grpc.DialContext(context.Background(),
		grpcAddress(cfg),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		t.Fatalf("failed to connect to gRPC server: %s", err)
	}

	privateKey, err := jwt.ReadPrivateKey()
	if err != nil {
		t.Fatal("failed to read private key")
	}
	publicKey, err := jwt.ReadPublicKey()
	if err != nil {
		t.Fatal("failed to read public key")
	}

	jwt := jwt.NewJWTClient(privateKey, publicKey, cfg.System.AccessTokenTimeout, cfg.System.RefreshTokenTimeout)

	return ctx, &Suite{
		T:                 t,
		Cfg:               &cfg,
		JWT:               jwt,
		AuthServiceClient: pb.NewAuthServiceClient(cc),
	}

}

func grpcAddress(cfg config.AppConfig) string {
	return net.JoinHostPort(grpcHost, cfg.GRPC.ListenAddress)
}
