-- +goose Up
CREATE TABLE users (
   id SERIAL PRIMARY KEY,
   name TEXT UNIQUE NOT NULL,
   password TEXT NOT NULL,
   created_at TIMESTAMP DEFAULT now(),
   updated_at TIMESTAMP DEFAULT now()
);

-- +goose Down
DROP TABLE users;