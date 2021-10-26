-- name: drop-table-cards

DROP TABLE IF EXISTS cards;

-- name: alter-table-steps-add-column-step_schema

ALTER TABLE steps
    ADD COLUMN step_schema VARCHAR(2000) NOT NULL DEFAULT '';

-- name: create-new-table-cards
CREATE TABLE IF NOT EXISTS cards
(
    card_id   INTEGER PRIMARY KEY,
    card_data BLOB,
    FOREIGN KEY (card_id) REFERENCES steps (step_id) ON DELETE CASCADE
);