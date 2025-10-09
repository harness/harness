-- Connectors table is not being used so can be dropped and recreated without
-- worrying about a migration
DROP TABLE IF EXISTS connectors;

CREATE TABLE connectors (
    -- Fields valid for all connectors
    connector_id SERIAL PRIMARY KEY,
    connector_identifier TEXT NOT NULL,
    connector_description TEXT NOT NULL,
    connector_type TEXT NOT NULL,
    connector_auth_type TEXT NOT NULL, -- basicauth, oidc, oauth, aws
    connector_created_by INTEGER NOT NULL,
    connector_space_id INTEGER NOT NULL,
    connector_last_test_attempt INTEGER NOT NULL,
    connector_last_test_error_msg TEXT NOT NULL,
    connector_last_test_status TEXT NOT NULL,
    connector_created BIGINT NOT NULL,
    connector_updated BIGINT NOT NULL,
    connector_version INTEGER NOT NULL,
    connector_address TEXT,
    connector_insecure BOOLEAN,

    -- Fields used by different connectors based on the auth_type
    connector_username TEXT,
    connector_github_app_installation_id TEXT,
    connector_github_app_application_id TEXT,
    connector_region TEXT,

    -- secrets (foreign keys to the secrets table and restricted on delete)
    connector_password INTEGER,
    connector_token INTEGER,
    connector_aws_key INTEGER,
    connector_aws_secret INTEGER,
    connector_github_app_private_key INTEGER,
    connector_token_refresh INTEGER,
    
    -- Foreign key to spaces table
    CONSTRAINT fk_connectors_space_id FOREIGN KEY (connector_space_id)
        REFERENCES spaces (space_id)
        ON UPDATE NO ACTION
        ON DELETE CASCADE,

    -- Foreign key to principals table
    CONSTRAINT fk_connectors_created_by FOREIGN KEY (connector_created_by)
        REFERENCES principals (principal_id)
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,

    -- Foreign key to secrets table
    CONSTRAINT fk_connectors_password FOREIGN KEY (connector_password)
        REFERENCES secrets (secret_id)
        ON UPDATE NO ACTION
        ON DELETE RESTRICT,

    -- Foreign key to secrets table
    CONSTRAINT fk_connectors_token FOREIGN KEY (connector_token)
        REFERENCES secrets (secret_id)
        ON UPDATE NO ACTION
        ON DELETE RESTRICT,

    -- Foreign key to secrets table
    CONSTRAINT fk_connectors_aws_key FOREIGN KEY (connector_aws_key)
        REFERENCES secrets (secret_id)
        ON UPDATE NO ACTION
        ON DELETE RESTRICT,

    -- Foreign key to secrets table
    CONSTRAINT fk_connectors_aws_secret FOREIGN KEY (connector_aws_secret)
        REFERENCES secrets (secret_id)
        ON UPDATE NO ACTION
        ON DELETE RESTRICT,

    -- Foreign key to secrets table
    CONSTRAINT fk_connectors_github_app_private_key FOREIGN KEY (connector_github_app_private_key)
        REFERENCES secrets (secret_id)
        ON UPDATE NO ACTION
        ON DELETE RESTRICT,

    -- Foreign key to secrets table
    CONSTRAINT fk_connectors_token_refresh FOREIGN KEY (connector_token_refresh)
        REFERENCES secrets (secret_id)
        ON UPDATE NO ACTION
        ON DELETE RESTRICT
);

-- Creating a unique index for case-insensitive connector identifiers
CREATE UNIQUE INDEX unique_connector_lowercase_identifier 
ON connectors(connector_space_id, LOWER(connector_identifier));
