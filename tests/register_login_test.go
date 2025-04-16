package tests

import (
	pb "github.com/HappyFreeman/gRPC-service-user/grpc/genproto"
	"github.com/HappyFreeman/gRPC-service-user/pkg/jwt"
	"github.com/HappyFreeman/gRPC-service-user/tests/suite"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
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

	//loginTime := time.Now()

	require.NoError(t, err)
	assert.NotEmpty(t, respLogin.GetAccessToken())

	token := respLogin.GetAccessToken()

	// TODO: tokenData добавить в ответ логина id и проверять его в token
	_, err = st.JWT.GetDataFromToken(&jwt.GetDataFromTokenParams{
		Token: token,
	})

	require.NoError(t, err)

	// assert.Equal(t, id, tokenData.UserId)
	//const deltaSeconds = 5
	//assert.InDelta(t, loginTime.Add(time.Hour).Unix(), tokenData.Exp, deltaSeconds)
}

func randomFakePassword() string {
	return gofakeit.Password(true, true, true, true, false, 10)
}
