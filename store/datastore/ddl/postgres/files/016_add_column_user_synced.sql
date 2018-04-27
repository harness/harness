-- name: alter-table-add-user-synced

ALTER TABLE users ADD COLUMN user_synced INTEGER;

-- name: update-table-set-user-synced

UPDATE users SET user_synced = 0;
