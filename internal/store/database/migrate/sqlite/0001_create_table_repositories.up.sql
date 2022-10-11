CREATE TABLE IF NOT EXISTS repositories (
 repo_id                INTEGER PRIMARY KEY AUTOINCREMENT
,repo_pathName          TEXT COLLATE NOCASE
,repo_spaceId           INTEGER
,repo_name              TEXT
,repo_description       TEXT
,repo_isPublic          BOOLEAN
,repo_createdBy         INTEGER
,repo_created           BIGINT
,repo_updated           BIGINT
,repo_forkId            INTEGER
,repo_numForks          INTEGER
,repo_numPulls          INTEGER
,repo_numClosedPulls    INTEGER
,repo_numOpenPulls      INTEGER
);
