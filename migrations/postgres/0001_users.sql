-- +goose Up
CREATE TABLE users (
   id SERIAL PRIMARY KEY,
   name TEXT UNIQUE NOT NULL,
   password TEXT NOT NULL,
   created_at TIMESTAMP DEFAULT now(),
   updated_at TIMESTAMP DEFAULT now() ON UPDATE now()
);

CREATE TABLE sessions (
  id SERIAL PRIMARY KEY,
  user_id INTEGER REFERENCES users(id) ON DELETE CASCADE NOT NULL,
  refresh_token TEXT UNIQUE NOT NULL,
  expiration_at TIMESTAMP NOT NULL
);

-- индекс для быстрого поиска по user_id
CREATE INDEX idx_sessions_user_id ON sessions(user_id);

-- +goose Down
DROP TABLE sessions;
DROP TABLE users;