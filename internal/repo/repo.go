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
	getUserByName   = `SELECT name, password FROM users WHERE name = $1;`
)

type repository struct {
	pool *pgxpool.Pool
}

// Repository - интерфейс с методами для юзера
type Repository interface {
	CreateUser(ctx context.Context, user User) (int, error)
	GetUserByName(ctx context.Context, name string) (User, error)
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

	err := r.pool.QueryRow(ctx, getUserByName, name).Scan(&user.Name, &user.Password)

	if err != nil {
		return User{}, errors.Wrap(err, "failed to get user by name")
	}

	return user, nil
}
