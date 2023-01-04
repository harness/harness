CREATE TABLE repositories (
 repo_id                INTEGER PRIMARY KEY AUTOINCREMENT
,repo_version           INTEGER NOT NULL
,repo_parent_id         INTEGER
,repo_uid               TEXT
,repo_description       TEXT
,repo_is_public         BOOLEAN NOT NULL
,repo_created_by        INTEGER NOT NULL
,repo_created           BIGINT NOT NULL
,repo_updated           BIGINT NOT NULL
,repo_git_uid           TEXT NOT NULL
,repo_default_branch    TEXT NOT NULL
,repo_fork_id           INTEGER
,repo_pullreq_seq       INTEGER NOT NULL
,repo_num_forks         INTEGER NOT NULL
,repo_num_pulls         INTEGER NOT NULL
,repo_num_closed_pulls  INTEGER NOT NULL
,repo_num_open_pulls    INTEGER NOT NULL

,UNIQUE(repo_git_uid)
);
