-- GetResource returns a resource matching the specified user and path.
-- name: GetResource :one
SELECT *
FROM resources
WHERE user = ?
  AND path = ?
LIMIT 1;

-- GetResources returns all resources matching the specified user and one of the paths.
-- name: GetResources :many
SELECT *
FROM resources
WHERE user = ?
  AND path in (sqlc.slice('paths'));

-- UpsertResource upserts a resource.
-- name: UpsertResource :one
INSERT INTO resources (user, path, hidden, starred, view_time, read_time)
VALUES (?1, ?2, ?3, ?4, ?5, ?6)
ON CONFLICT DO UPDATE SET hidden    = ?3,
                          starred   = ?4,
                          view_time = ?5,
                          read_time = ?6
WHERE user = ?1
  AND path = ?2
RETURNING *;

-- UpsertUser adds a new user with the given name and password_hash.
-- name: UpsertUser :one
INSERT INTO users (id, name, password_hash)
VALUES (?1, ?2, ?3)
ON CONFLICT DO UPDATE SET name          = ?2,
                          password_hash = ?3
WHERE id = ?1
RETURNING *;

-- GetUser fetches a user by ID.
-- name: GetUser :one
SELECT *
FROM users
WHERE id = ?
LIMIT 1;

-- GetUserByName fetches a user by their name.
-- name: GetUserByName :one
SELECT *
FROM users
WHERE name = ?
LIMIT 1;

-- GetUsers fetches a list of users starting after the provided name.
-- name: GetUsers :many
SELECT *
FROM users
WHERE name > sqlc.arg(after_name)
ORDER BY name
LIMIT ?1;

-- SetUserName updates a user's name with the given ID.
-- name: SetUserName :one
UPDATE users
SET name = ?2
WHERE id = ?1
RETURNING *;

-- SetUserPasswordHash updates a user's password hash with the given ID.
-- name: SetUserPasswordHash :one
UPDATE users
SET password_hash = ?2
WHERE id = ?1
RETURNING *;

-- DeleteUser removes a user and their resources from the system.
-- name: DeleteUser :exec
DELETE
FROM users
WHERE id = ?;
