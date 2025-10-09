CREATE TABLE rules (
 rule_id SERIAL PRIMARY KEY
,rule_version INTEGER NOT NULL
,rule_created_by INTEGER NOT NULL
,rule_created BIGINT NOT NULL
,rule_updated BIGINT NOT NULL
,rule_space_id INTEGER
,rule_repo_id INTEGER
,rule_uid TEXT NOT NULL
,rule_description TEXT NOT NULL
,rule_type TEXT NOT NULL
,rule_state TEXT NOT NULL
,rule_pattern JSON NOT NULL
,rule_definition JSON NOT NULL
,CONSTRAINT fk_rule_created_by FOREIGN KEY (rule_created_by)
    REFERENCES principals (principal_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE SET NULL
,CONSTRAINT fk_rule_space_id FOREIGN KEY (rule_space_id)
    REFERENCES spaces (space_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
,CONSTRAINT fk_rule_repo_id FOREIGN KEY (rule_repo_id)
    REFERENCES repositories (repo_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
);

CREATE UNIQUE INDEX rules_space_id_uid
	ON rules(rule_space_id, LOWER(rule_uid))
	WHERE rule_space_id IS NOT NULL;

CREATE UNIQUE INDEX rules_repo_id_uid
    ON rules(rule_repo_id, LOWER(rule_uid))
    WHERE rule_repo_id IS NOT NULL;
