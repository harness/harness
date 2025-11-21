CREATE TABLE IF NOT EXISTS favorite_repos
(
    favorite_principal_id INTEGER NOT NULL,
    favorite_repo_id      INTEGER NOT NULL,
    favorite_created      BIGINT NOT NULL,
    PRIMARY KEY (favorite_principal_id, favorite_repo_id),
    CONSTRAINT fk_frepos_principal_id FOREIGN KEY (favorite_principal_id)
        REFERENCES principals (principal_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_frepos_repo_id FOREIGN KEY (favorite_repo_id)
        REFERENCES repositories (repo_id)
        ON DELETE CASCADE
);