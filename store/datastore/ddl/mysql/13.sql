-- +migrate Up

CREATE TABLE procs (
 proc_id         INTEGER PRIMARY KEY AUTO_INCREMENT
,proc_build_id   INTEGER
,proc_pid        INTEGER
,proc_ppid       INTEGER
,proc_pgid       INTEGER
,proc_name       VARCHAR(250)
,proc_state      VARCHAR(250)
,proc_error      VARCHAR(500)
,proc_exit_code  INTEGER
,proc_started    INTEGER
,proc_stopped    INTEGER
,proc_machine    VARCHAR(250)
,proc_platform   VARCHAR(250)
,proc_environ    VARCHAR(2000)
,UNIQUE(proc_build_id, proc_pid)
);

CREATE INDEX proc_build_ix ON procs (proc_build_id);

CREATE TABLE files (
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

CREATE INDEX file_build_ix ON files (file_build_id);
CREATE INDEX file_proc_ix  ON files (file_proc_id);

-- +migrate Down

DROP INDEX file_build_ix;
DROP INDEX file_proc_ix;
DROP TABLE files;

DROP INDEX proc_build_ix;
DROP TABLE procs;
