-- name: create-table-files

CREATE TABLE IF NOT EXISTS files (
 file_id       INTEGER PRIMARY KEY AUTOINCREMENT
,file_build_id INTEGER
,file_proc_id  INTEGER
,file_name     TEXT
,file_mime     TEXT
,file_size     INTEGER
,file_time     INTEGER
,file_data     BLOB
,UNIQUE(file_proc_id,file_name)
);

-- name: create-index-files-builds

CREATE INDEX IF NOT EXISTS file_build_ix ON files (file_build_id);

-- name: create-index-files-procs

CREATE INDEX IF NOT EXISTS file_proc_ix  ON files (file_proc_id);
