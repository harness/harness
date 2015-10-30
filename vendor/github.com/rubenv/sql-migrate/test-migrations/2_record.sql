-- +migrate Up
INSERT INTO people (id) VALUES (1);

-- +migrate Down
DELETE FROM people WHERE id=1;
