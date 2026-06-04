ALTER TABLE pullreqs ADD COLUMN pullreq_type TEXT;

CREATE TABLE linked_pullreqs (
 linked_pullreq_id INTEGER NOT NULL
,linked_pullreq_provider_type TEXT NOT NULL
,linked_pullreq_provider_repo_id TEXT NOT NULL
,linked_pullreq_provider_url TEXT NOT NULL
,linked_pullreq_provider_author_login TEXT NOT NULL
,linked_pullreq_provider_author_avatar_url TEXT NOT NULL DEFAULT ''
,linked_pullreq_provider_author_url TEXT NOT NULL DEFAULT ''
,linked_pullreq_last_synced_at BIGINT NOT NULL
,linked_pullreq_provider_updated_at BIGINT NOT NULL DEFAULT 0
,linked_pullreq_merger_login TEXT NOT NULL DEFAULT ''
,CONSTRAINT pk_linked_pullreqs PRIMARY KEY (linked_pullreq_id)
,CONSTRAINT fk_linked_pullreqs_pullreq_id FOREIGN KEY (linked_pullreq_id)
    REFERENCES pullreqs (pullreq_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
);

-- Secondary index for inbound-webhook routing. PR number lives on the parent
-- pullreqs row and is matched via JOIN; (provider_type, provider_repo_id)
-- alone is not unique because multiple linked_repositories rows may mirror
-- the same upstream repo.
CREATE INDEX idx_linked_pullreqs_provider_id
    ON linked_pullreqs (linked_pullreq_provider_type, linked_pullreq_provider_repo_id);
