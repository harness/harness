-- name: create-table-procs

CREATE TABLE IF NOT EXISTS procs (
 proc_id         SERIAL PRIMARY KEY
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

-- name: create-index-procs-build

CREATE INDEX IF NOT EXISTS proc_build_ix ON procs (proc_build_id);
