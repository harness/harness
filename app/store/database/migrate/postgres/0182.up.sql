CREATE TABLE pullreq_file_groups (
 pullreq_file_group_id BIGSERIAL PRIMARY KEY
,pullreq_file_group_pr_id INTEGER NOT NULL
,pullreq_file_group_title TEXT NOT NULL
,pullreq_file_group_description TEXT NOT NULL DEFAULT ''
,pullreq_file_group_created BIGINT NOT NULL
,pullreq_file_group_updated BIGINT NOT NULL
,pullreq_file_group_created_by INTEGER NOT NULL
,pullreq_file_group_updated_by INTEGER NOT NULL

,CONSTRAINT uq_pullreq_file_group_pr_id_title UNIQUE (pullreq_file_group_pr_id, pullreq_file_group_title)
,CONSTRAINT fk_pullreq_file_group_pr_id FOREIGN KEY (pullreq_file_group_pr_id)
    REFERENCES pullreqs (pullreq_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
,CONSTRAINT fk_pullreq_file_group_created_by FOREIGN KEY (pullreq_file_group_created_by)
    REFERENCES principals (principal_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE NO ACTION
,CONSTRAINT fk_pullreq_file_group_updated_by FOREIGN KEY (pullreq_file_group_updated_by)
    REFERENCES principals (principal_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE NO ACTION
);

CREATE TABLE pullreq_file_group_files (
 pullreq_file_group_file_pullreq_file_group_id BIGINT NOT NULL
,pullreq_file_group_file_pr_id INTEGER NOT NULL
,pullreq_file_group_file_path TEXT NOT NULL
,pullreq_file_group_file_old_sha TEXT
,pullreq_file_group_file_new_sha TEXT

,CONSTRAINT pk_pullreq_file_group_files PRIMARY KEY (
    pullreq_file_group_file_pr_id,
    pullreq_file_group_file_path
)
,CONSTRAINT fk_pullreq_file_group_file_pr_id FOREIGN KEY (pullreq_file_group_file_pr_id)
    REFERENCES pullreqs (pullreq_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
,CONSTRAINT fk_pullreq_file_group_file_pullreq_file_group_id FOREIGN KEY (pullreq_file_group_file_pullreq_file_group_id)
    REFERENCES pullreq_file_groups (pullreq_file_group_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
);