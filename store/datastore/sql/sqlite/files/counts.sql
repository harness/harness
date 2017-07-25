-- name: count-users

SELECT count(1)
FROM users

-- name: count-repos

SELECT count(1)
FROM repos
WHERE repo_active = 1

-- name: count-builds

SELECT count(1)
FROM builds
