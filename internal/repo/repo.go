package repo

import (
	"context"
	"fmt"
	"github.com/HappyFreeman/gRPC-service-user/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

// Слой репозитория, здесь должны быть все методы, связанные с базой данных

const (
	insertUserQuery = `INSERT INTO users (name, password) VALUES ($1, $2) RETURNING id;`
	getUserByName   = `SELECT id, name, password FROM users WHERE name = $1;`
	changePassword  = `UPDATE users SET password = $2, updated_at = now() WHERE id = $1;`

	insertRefreshTokenQuery = `INSERT INTO sessions (user_id, refresh_token, expiration_at) VALUES ($1, $2, $3) RETURNING id;`
	getRefreshTokenQuery    = "SELECT refresh_token FROM sessions WHERE user_id = $1;"
	updateRefreshTokenQuery = "UPDATE sessions SET refresh_token = $1, expiration_at = $2 WHERE user_id = $3;"
)

type repository struct {
	pool *pgxpool.Pool
}

// Repository - интерфейс с методами для юзера
type Repository interface {
	CreateUser(ctx context.Context, user User) (int, error)
	GetUserByName(ctx context.Context, name string) (User, error)
	ChangePassword(ctx context.Context, params UpdatePasswordParams) error

	// Методы работы с токенами.
	NewRefreshToken(ctx context.Context, params NewRefreshTokenParams) (int64, error)
	GetRefreshToken(ctx context.Context, params GetRefreshTokenParams) ([]string, error)
	UpdateRefreshToken(ctx context.Context, params UpdateRefreshTokenParams) error
}

func NewRepository(ctx context.Context, cfg config.PostgreSQL) (Repository, error) {
	// Формируем строку подключения
	connString := fmt.Sprintf(
		`user=%s password=%s host=%s port=%d dbname=%s sslmode=%s 
        pool_max_conns=%d pool_max_conn_lifetime=%s pool_max_conn_idle_time=%s`,
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
		cfg.SSLMode,
		cfg.PoolMaxConns,
		cfg.PoolMaxConnLifetime.String(),
		cfg.PoolMaxConnIdleTime.String(),
	)

	// Парсим конфигурацию подключения
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse PostgreSQL config")
	}

	// Оптимизация выполнения запросов (кеширование запросов)
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheDescribe

	// Создаём пул соединений с базой данных
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create PostgreSQL connection pool")
	}

	return &repository{pool}, nil
}

func (r *repository) CreateUser(ctx context.Context, user User) (int, error) {
	var id int
	err := r.pool.QueryRow(ctx, insertUserQuery, user.Name, user.Password).Scan(&id)

	if err != nil {
		return 0, errors.Wrap(err, "failed to create user")
	}

	return id, err
}

func (r *repository) GetUserByName(ctx context.Context, name string) (User, error) {
	var user User

	err := r.pool.QueryRow(ctx, getUserByName, name).Scan(&user.ID, &user.Name, &user.Password)

	if err != nil {
		return User{}, errors.Wrap(err, "failed to get user by name")
	}

	return user, nil
}

func (r *repository) ChangePassword(ctx context.Context, params UpdatePasswordParams) error {

	_, err := r.pool.Exec(ctx, changePassword, params.UserID, params.Password)

	if err != nil {
		return errors.Wrap(err, "failed to change password")
	}

	return nil
}

func (r *repository) NewRefreshToken(ctx context.Context, params NewRefreshTokenParams) (int64, error) {
	var id int64

	err := r.pool.QueryRow(ctx, insertRefreshTokenQuery, params.UserID, params.Token, params.ExpirationAt).Scan(&id)

	if err != nil {
		return 0, errors.Wrap(err, "failed to create refresh token")
	}

	return id, nil
}

func (r *repository) GetRefreshToken(ctx context.Context, params GetRefreshTokenParams) ([]string, error) {
	rows, err := r.pool.Query(ctx, getRefreshTokenQuery, params.UserID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get refresh token")
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, errors.Wrap(err, "failed to scan refresh token")
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

func (r *repository) UpdateRefreshToken(ctx context.Context, params UpdateRefreshTokenParams) error {
	_, err := r.pool.Exec(ctx, updateRefreshTokenQuery, params.Token, params.ExpirationAt, params.UserID)
	if err != nil {
		return errors.Wrap(err, "failed to update refresh token")
	}
	return nil
}
