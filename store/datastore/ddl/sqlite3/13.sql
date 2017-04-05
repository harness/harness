-- +migrate Up

CREATE TABLE procs (
 proc_id         INTEGER PRIMARY KEY AUTOINCREMENT
,proc_build_id   INTEGER
,proc_pid        INTEGER
,proc_ppid       INTEGER
,proc_pgid       INTEGER
,proc_name       TEXT
,proc_state      TEXT
,proc_error      TEXT
,proc_exit_code  INTEGER
,proc_started    INTEGER
,proc_stopped    INTEGER
,proc_machine    TEXT
,proc_platform   TEXT
,proc_environ    TEXT
,UNIQUE(proc_build_id, proc_pid)
);

CREATE INDEX proc_build_ix ON procs (proc_build_id);

CREATE TABLE files (
 file_id       INTEGER PRIMARY KEY AUTOINCREMENT
,file_build_id INTEGER
,file_proc_id  INTEGER
,file_name     TEXT
,file_mime     TEXT
,file_size     INTEGER
,file_time     INTEGER
,file_data     BLOB
,UNIQUE(file_proc_id,file_name)
,FOREIGN KEY(file_proc_id) REFERENCES procs (proc_id) ON DELETE CASCADE
);

CREATE INDEX file_build_ix ON files (file_build_id);
CREATE INDEX file_proc_ix  ON files (file_proc_id);

-- +migrate Down

DROP INDEX file_build_ix;
DROP INDEX file_proc_ix;
DROP TABLE files;

DROP INDEX proc_build_ix;
DROP TABLE procs;
