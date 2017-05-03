-- name: count-users

SELECT reltuples
FROM pg_class WHERE relname = 'users';

-- name: count-repos

SELECT reltuples
FROM pg_class WHERE relname = 'repos';

-- name: count-builds

SELECT reltuples
FROM pg_class WHERE relname = 'builds';
