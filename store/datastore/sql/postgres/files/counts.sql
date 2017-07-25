-- name: count-users

SELECT reltuples
FROM pg_class WHERE relname = 'users'

-- name: count-repos

SELECT count(1)
FROM repo
WHERE repo_active = 1

-- name: count-builds

SELECT reltuples
FROM pg_class WHERE relname = 'builds'
