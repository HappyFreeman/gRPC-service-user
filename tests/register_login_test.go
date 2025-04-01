package tests

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	pb "github.com/HappyFreeman/gRPC-service-user/internal/proto/gen"
	"github.com/HappyFreeman/gRPC-service-user/tests/suite"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"
)

const (
	secret = "secret"
)

func TestRegisterLogin_Login_HappyPath(t *testing.T) {
	ctx, st := suite.New(t)

	name := gofakeit.Name()
	pass := randomFakePassword()

	respReg, err := st.AuthServiceClient.Register(ctx, &pb.RegisterRequest{
		Username: name,
		Password: pass,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, respReg.GetMessage())

	respLogin, err := st.AuthServiceClient.Login(ctx, &pb.LoginRequest{
		Username: name,
		Password: pass,
	})

	loginTime := time.Now()

	require.NoError(t, err)
	assert.NotEmpty(t, respLogin.GetToken())

	token := respLogin.GetToken()

	tokenParse, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	require.NoError(t, err)

	claims, ok := tokenParse.Claims.(jwt.MapClaims)

	assert.True(t, ok)
	assert.Equal(t, name, claims["name"].(string))

	const deltaSeconds = 5

	assert.InDelta(t, loginTime.Add(time.Hour).Unix(), claims["exp"].(float64), deltaSeconds)
}

func randomFakePassword() string {
	return gofakeit.Password(true, true, true, true, false, 10)
}
