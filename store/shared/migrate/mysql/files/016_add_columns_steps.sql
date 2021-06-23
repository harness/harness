-- name: alter-table-steps-add-column-step-depends-on

ALTER TABLE steps ADD COLUMN step_depends_on TEXT NULL;

-- name: alter-table-steps-add-column-step-image

ALTER TABLE steps ADD COLUMN step_image VARCHAR(1000) NOT NULL DEFAULT '';

-- name: alter-table-steps-add-column-step-detached

ALTER TABLE steps ADD COLUMN step_detached BOOLEAN NOT NULL DEFAULT FALSE;
