package repo

import "database/sql"

type User struct {
	ID       int64  `json:"id"`
	Name     string `json:"name" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type NewRefreshTokenParams struct {
	UserID       int64
	Token        string
	ExpirationAt sql.NullTime
}

type GetRefreshTokenParams struct {
	UserID int64
}

type UpdateRefreshTokenParams struct {
	UserID       int64
	Token        string
	ExpirationAt sql.NullTime
}

type UpdatePasswordParams struct {
	UserID   int64
	Password string
}
