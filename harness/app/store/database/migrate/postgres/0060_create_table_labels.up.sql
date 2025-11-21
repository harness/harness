CREATE TABLE labels (
    label_id SERIAL PRIMARY KEY,
    label_space_id INTEGER DEFAULT NULL,
    label_repo_id INTEGER DEFAULT NULL,
    label_scope INTEGER DEFAULT 0,
    label_key TEXT NOT NULL,
    label_description TEXT NOT NULL DEFAULT '',
    label_color TEXT NOT NULL DEFAULT 'black',
    label_type TEXT NOT NULL DEFAULT 'static',
    label_created BIGINT NOT NULL,
    label_updated BIGINT NOT NULL,
    label_created_by INTEGER NOT NULL,
    label_updated_by INTEGER NOT NULL,
    label_value_count INTEGER DEFAULT 0,

    CONSTRAINT fk_labels_space_id FOREIGN KEY (label_space_id)
        REFERENCES spaces (space_id) ON DELETE CASCADE,
    CONSTRAINT fk_labels_repo_id FOREIGN KEY (label_repo_id)
        REFERENCES repositories (repo_id) ON DELETE CASCADE,
    CONSTRAINT chk_label_space_or_repo 
        CHECK (label_space_id IS NULL OR label_repo_id IS NULL),
    CONSTRAINT fk_labels_created_by FOREIGN KEY (label_created_by)
        REFERENCES principals (principal_id),
    CONSTRAINT fk_labels_updated_by FOREIGN KEY (label_updated_by)
        REFERENCES principals (principal_id)
);

CREATE UNIQUE INDEX labels_space_id_key
ON labels(label_space_id, LOWER(label_key))
WHERE label_space_id IS NOT NULL;

CREATE UNIQUE INDEX labels_repo_id_key
ON labels(label_repo_id, LOWER(label_key))
WHERE label_repo_id IS NOT NULL;

CREATE TABLE label_values (
    label_value_id SERIAL PRIMARY KEY,
    label_value_label_id INTEGER NOT NULL,
    label_value_value TEXT NOT NULL,
    label_value_color TEXT NOT NULL,
    label_value_created BIGINT NOT NULL,
    label_value_updated BIGINT NOT NULL,
    label_value_created_by INTEGER NOT NULL,
    label_value_updated_by INTEGER NOT NULL,

    CONSTRAINT fk_label_values_label_id FOREIGN KEY (label_value_label_id)
        REFERENCES labels (label_id) ON DELETE CASCADE,
    CONSTRAINT fk_label_values_created_by FOREIGN KEY (label_value_created_by)
        REFERENCES principals (principal_id),
    CONSTRAINT fk_labels_values_updated_by FOREIGN KEY (label_value_updated_by)
        REFERENCES principals (principal_id)
);

CREATE UNIQUE INDEX unique_label_value_label_id_value
ON label_values(label_value_label_id, LOWER(label_value_value));

CREATE TABLE pullreq_labels (
    pullreq_label_pullreq_id INTEGER NOT NULL,
    pullreq_label_label_id INTEGER NOT NULL,
    pullreq_label_label_value_id INTEGER DEFAULT NULL,
    pullreq_label_created BIGINT NOT NULL,
    pullreq_label_updated BIGINT NOT NULL,
    pullreq_label_created_by INTEGER NOT NULL,
    pullreq_label_updated_by INTEGER NOT NULL,

    CONSTRAINT fk_pullreq_labels_pullreq_id FOREIGN KEY (pullreq_label_pullreq_id)
        REFERENCES pullreqs (pullreq_id) ON DELETE CASCADE,
    CONSTRAINT fk_pullreq_labels_label_id FOREIGN KEY (pullreq_label_label_id)
        REFERENCES labels (label_id) ON DELETE CASCADE,
    CONSTRAINT fk_pullreq_labels_label_value_id FOREIGN KEY (pullreq_label_label_value_id)
        REFERENCES label_values (label_value_id) ON DELETE SET NULL,
    CONSTRAINT fk_pullreq_labels_created_by FOREIGN KEY (pullreq_label_created_by)
        REFERENCES principals (principal_id),
    CONSTRAINT fk_pullreq_labels_updated_by FOREIGN KEY (pullreq_label_updated_by)
        REFERENCES principals (principal_id),

    PRIMARY KEY (pullreq_label_pullreq_id, pullreq_label_label_id)
);
