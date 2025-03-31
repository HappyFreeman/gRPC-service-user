package repo

type User struct {
	Name     string `json:"name" validate:"required"`
	Password string `json:"password" validate:"required"`
}
