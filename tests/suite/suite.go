package suite

import (
	"context"
	"github.com/HappyFreeman/gRPC-service-user/internal/config"
	pb "github.com/HappyFreeman/gRPC-service-user/internal/proto/gen"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net"
	"testing"
)

const (
	grpcHost = "localhost"
)

type Suite struct {
	*testing.T
	Cfg               *config.AppConfig
	AuthServiceClient pb.AuthServiceClient
}

func New(t *testing.T) (context.Context, *Suite) {
	t.Helper()
	t.Parallel()

	// Загружаем env в переменные окружения
	if err := godotenv.Load("../.env"); err != nil {
		log.Fatal(errors.Wrap(err, "failed to load environment variables"))
	}

	// Загружаем конфигурацию из переменных окружения
	var cfg config.AppConfig
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatal(errors.Wrap(err, "failed to load configuration"))
	}

	ctx, cancelCtx := context.WithTimeout(context.Background(), cfg.GRPC.WriteTimeout)

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

	return ctx, &Suite{
		T:                 t,
		Cfg:               &cfg,
		AuthServiceClient: pb.NewAuthServiceClient(cc),
	}

}

func grpcAddress(cfg config.AppConfig) string {
	return net.JoinHostPort(grpcHost, cfg.GRPC.ListenAddress)
}
