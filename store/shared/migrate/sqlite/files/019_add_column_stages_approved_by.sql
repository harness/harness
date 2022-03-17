-- name: alter-table-stages-add-column-approved-by

ALTER TABLE stages
    ADD COLUMN stage_approved_by TEXT NOT NULL DEFAULT '';
