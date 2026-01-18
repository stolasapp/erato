-- +goose Up
-- +goose StatementBegin
PRAGMA encoding = 'UTF-8';

CREATE TABLE IF NOT EXISTS users
(
    id            BIGINT NOT NULL PRIMARY KEY,
    name          TEXT   NOT NULL UNIQUE,
    password_hash BLOB   NOT NULL
);

CREATE TABLE IF NOT EXISTS resources
(
    user      BIGINT    NOT NULL,
    path      TEXT      NOT NULL,
    hidden    BOOLEAN   NOT NULL DEFAULT false,
    starred   BOOLEAN   NOT NULL DEFAULT false,
    view_time TIMESTAMP NULL,
    read_time TIMESTAMP NULL,
    PRIMARY KEY (user, path),
    FOREIGN KEY(user)
      REFERENCES users(id)
      ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS resources;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
