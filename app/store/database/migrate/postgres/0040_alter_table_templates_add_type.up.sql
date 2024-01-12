DROP TABLE IF exists templates;

CREATE TABLE templates (
    template_id SERIAL PRIMARY KEY
    ,template_uid TEXT NOT NULL
    ,template_type TEXT NOT NULL
    ,template_description TEXT NOT NULL
    ,template_space_id INTEGER NOT NULL
    ,template_data BYTEA NOT NULL
    ,template_created BIGINT NOT NULL
    ,template_updated BIGINT NOT NULL
    ,template_version INTEGER NOT NULL

    -- Ensure unique combination of space ID, UID and template type
    ,UNIQUE (template_space_id, template_uid, template_type)

    -- Foreign key to spaces table
    ,CONSTRAINT fk_templates_space_id FOREIGN KEY (template_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE
);