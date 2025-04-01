package jwt

import (
	"github.com/HappyFreeman/gRPC-service-user/internal/repo"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

func NewToken(user repo.User, duration time.Duration) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	claims["exp"] = time.Now().Add(duration).Unix()
	claims["name"] = user.Name

	// Подписываем токен
	//TODO: вынести в конфиг secret
	tokenString, err := token.SignedString([]byte("secret"))

	if err != nil {
		return "", err
	}

	return tokenString, nil
}
