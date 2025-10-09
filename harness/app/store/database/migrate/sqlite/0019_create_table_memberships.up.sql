CREATE TABLE memberships (
 membership_space_id INTEGER NOT NULL
,membership_principal_id INTEGER NOT NULL
,membership_created_by INTEGER NOT NULL
,membership_created BIGINT NOT NULL
,membership_updated BIGINT NOT NULL
,membership_role TEXT NOT NULL
,CONSTRAINT pk_memberships PRIMARY KEY (membership_space_id, membership_principal_id)
,CONSTRAINT fk_membership_space_id FOREIGN KEY (membership_space_id)
    REFERENCES spaces (space_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
,CONSTRAINT fk_membership_principal_id FOREIGN KEY (membership_principal_id)
    REFERENCES principals (principal_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
,CONSTRAINT fk_membership_created_by FOREIGN KEY (membership_created_by)
    REFERENCES principals (principal_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE NO ACTION
);
