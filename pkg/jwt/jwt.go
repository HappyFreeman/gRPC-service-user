package jwt

import (
	"errors"
	"github.com/HappyFreeman/gRPC-service-user/internal/config"
	"github.com/HappyFreeman/gRPC-service-user/internal/repo"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

func NewToken(user repo.User, cfgJWT config.JWT) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	claims["exp"] = time.Now().Add(cfgJWT.TTL).Unix()
	claims["id"] = user.ID
	claims["name"] = user.Name

	// Подписываем токен
	tokenString, err := token.SignedString([]byte(cfgJWT.Secret))

	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GetUserId Получение ID пользователя из токена
func GetUserId(tokenString string, secret string) (int, error) {
	claims, err := validateToken(tokenString, secret)
	if err != nil {
		return 0, err
	}

	// Читаем ID пользователя
	idFloat, ok := claims["id"].(float64)
	if !ok {
		return 0, errors.New("invalid token id type")
	}

	return int(idFloat), nil
}

// validateToken Приватный метод для проверки токена
func validateToken(tokenString string, secret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Проверяем, что метод подписи — HMAC SHA256
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	// Проверяем claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	// Проверяем срок действия токена
	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, errors.New("invalid token exp type")
	}
	if time.Now().Unix() > int64(exp) {
		return nil, errors.New("token expired")
	}

	return claims, nil
}
