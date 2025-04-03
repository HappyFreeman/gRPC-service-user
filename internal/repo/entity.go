package repo

type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name" validate:"required"`
	Password string `json:"password" validate:"required"`
}
