ALTER TABLE repositories ADD COLUMN repo_type TEXT;

CREATE TABLE linked_repositories (
 linked_repo_id INTEGER NOT NULL
,linked_repo_version INTEGER NOT NULL
,linked_repo_created BIGINT NOT NULL
,linked_repo_updated BIGINT NOT NULL
,linked_repo_last_full_sync BIGINT NOT NULL
,linked_repo_connector_path TEXT NOT NULL
,linked_repo_connector_identifier TEXT NOT NULL
,linked_repo_connector_repo TEXT NOT NULL
,CONSTRAINT pk_linked_repositories PRIMARY KEY (linked_repo_id)
,CONSTRAINT fk_linked_repositories_repo_id FOREIGN KEY (linked_repo_id)
    REFERENCES repositories (repo_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
);
