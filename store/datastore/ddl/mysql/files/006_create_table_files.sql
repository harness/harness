-- name: create-table-files

CREATE TABLE IF NOT EXISTS files (
 file_id       INTEGER PRIMARY KEY AUTO_INCREMENT
,file_build_id INTEGER
,file_proc_id  INTEGER
,file_name     VARCHAR(250)
,file_mime     VARCHAR(250)
,file_size     INTEGER
,file_time     INTEGER
,file_data     MEDIUMBLOB

,UNIQUE(file_proc_id,file_name)
);

-- name: create-index-files-builds

CREATE INDEX file_build_ix ON files (file_build_id);

-- name: create-index-files-procs

CREATE INDEX file_proc_ix  ON files (file_proc_id);
