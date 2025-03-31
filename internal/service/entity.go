package service

// UserRequest - структура, представляющая тело запроса
type UserRequest struct {
	Name     string `json:"name" validate:"required"`
	Password string `json:"password" validate:"required"`
}
