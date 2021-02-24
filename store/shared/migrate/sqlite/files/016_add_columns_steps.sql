-- name: alter-table-steps-add-column-step-depends-on

ALTER TABLE steps ADD COLUMN step_depends_on TEXT NOT NULL DEFAULT '';

-- name: alter-table-steps-add-column-step-image

ALTER TABLE steps ADD COLUMN step_image TEXT NOT NULL DEFAULT '';

-- name: alter-table-steps-add-column-step-detached

ALTER TABLE steps ADD COLUMN step_detached BOOLEAN NOT NULL DEFAULT FALSE;
