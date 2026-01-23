CREATE TABLE autolinks (
    autolink_id INTEGER PRIMARY KEY AUTOINCREMENT,
    autolink_repo_id INTEGER DEFAULT NULL,
    autolink_space_id INTEGER DEFAULT NULL,
    autolink_type TEXT NOT NULL,
    autolink_pattern TEXT NOT NULL,
    autolink_target_url TEXT NOT NULL,
    autolink_created BIGINT NOT NULL,
    autolink_updated BIGINT NOT NULL,
    autolink_created_by INTEGER NOT NULL,
    autolink_updated_by INTEGER NOT NULL,

    CONSTRAINT fk_autolinks_repo_id FOREIGN KEY (autolink_repo_id)
        REFERENCES repositories (repo_id)
        ON DELETE CASCADE
        ON UPDATE NO ACTION,

    CONSTRAINT fk_autolinks_space_id FOREIGN KEY (autolink_space_id)
        REFERENCES spaces (space_id)
        ON DELETE CASCADE
        ON UPDATE NO ACTION,

    CONSTRAINT chk_autolink_repo_or_space
        CHECK (autolink_repo_id IS NULL OR autolink_space_id IS NULL),

    CONSTRAINT fk_autolinks_created_by FOREIGN KEY (autolink_created_by)
        REFERENCES principals (principal_id)
        ON DELETE NO ACTION
        ON UPDATE NO ACTION,

    CONSTRAINT fk_autolinks_updated_by FOREIGN KEY (autolink_updated_by)
        REFERENCES principals (principal_id)
        ON DELETE NO ACTION
        ON UPDATE NO ACTION
);

CREATE INDEX idx_autolinks_repo_id ON autolinks(autolink_repo_id) WHERE autolink_repo_id IS NOT NULL;
CREATE INDEX idx_autolinks_space_id ON autolinks(autolink_space_id) WHERE autolink_space_id IS NOT NULL;