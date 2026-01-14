ALTER TABLE pullreqs
    DROP CONSTRAINT fk_pullreq_source_repo_id;

ALTER TABLE pullreqs
    ADD CONSTRAINT fk_pullreq_source_repo_id FOREIGN KEY (pullreq_source_repo_id)
        REFERENCES repositories (repo_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE;