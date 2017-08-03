-- name: alter-table-add-file-pid

ALTER TABLE files ADD COLUMN file_pid INTEGER

-- name: alter-table-add-file-meta-passed

ALTER TABLE files ADD COLUMN file_meta_passed INTEGER

-- name: alter-table-add-file-meta-failed

ALTER TABLE files ADD COLUMN file_meta_failed INTEGER

-- name: alter-table-add-file-meta-skipped

ALTER TABLE files ADD COLUMN file_meta_skipped INTEGER

-- name: alter-table-update-file-meta

UPDATE files SET
 file_meta_passed=0
,file_meta_failed=0
,file_meta_skipped=0
