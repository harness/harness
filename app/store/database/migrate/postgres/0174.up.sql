CREATE TABLE merge_queues (
	 merge_queue_id              SERIAL PRIMARY KEY
	,merge_queue_repo_id         INTEGER NOT NULL
	,merge_queue_branch          TEXT NOT NULL
	,merge_queue_version         INTEGER NOT NULL
	,merge_queue_created         BIGINT NOT NULL
	,merge_queue_updated         BIGINT NOT NULL
	,merge_queue_order_sequence  INTEGER NOT NULL

	,CONSTRAINT fk_merge_queues_repo_id FOREIGN KEY (merge_queue_repo_id)
		REFERENCES repositories (repo_id) MATCH SIMPLE
		ON UPDATE NO ACTION
		ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_merge_queues_repo_id_branch_name ON merge_queues (merge_queue_repo_id, merge_queue_branch);

CREATE TABLE merge_queue_entries (
	 merge_queue_entry_pullreq_id          INTEGER NOT NULL
	,merge_queue_entry_queue_id            INTEGER NOT NULL
	,merge_queue_entry_version             INTEGER NOT NULL
	,merge_queue_entry_created_by          INTEGER NOT NULL
	,merge_queue_entry_created             BIGINT NOT NULL
	,merge_queue_entry_updated             BIGINT NOT NULL
	,merge_queue_entry_order_index         INTEGER NOT NULL
	,merge_queue_entry_state               TEXT NOT NULL
	,merge_queue_entry_base_commit_sha     TEXT
	,merge_queue_entry_head_commit_sha     TEXT
	,merge_queue_entry_merge_commit_sha    TEXT
	,merge_queue_entry_merge_base_sha      TEXT
	,merge_queue_entry_commit_count        INTEGER NOT NULL
	,merge_queue_entry_changed_file_count  INTEGER NOT NULL
	,merge_queue_entry_additions           INTEGER NOT NULL
	,merge_queue_entry_deletions           INTEGER NOT NULL
	,merge_queue_entry_checks_commit_sha   TEXT
	,merge_queue_entry_checks_started      BIGINT
	,merge_queue_entry_checks_deadline     BIGINT
	,merge_queue_entry_merge_method        TEXT NOT NULL
	,merge_queue_entry_commit_title        TEXT NOT NULL
	,merge_queue_entry_commit_message      TEXT NOT NULL
	,merge_queue_entry_delete_source_branch BOOLEAN NOT NULL

	,CONSTRAINT pk_merge_queue_entries PRIMARY KEY (merge_queue_entry_pullreq_id)
	,CONSTRAINT fk_merge_queue_entries_pullreq_id FOREIGN KEY (merge_queue_entry_pullreq_id)
		REFERENCES pullreqs (pullreq_id) MATCH SIMPLE
		ON UPDATE NO ACTION
		ON DELETE NO ACTION
	,CONSTRAINT fk_merge_queue_entries_queue_id FOREIGN KEY (merge_queue_entry_queue_id)
		REFERENCES merge_queues (merge_queue_id) MATCH SIMPLE
		ON UPDATE NO ACTION
		ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_merge_queue_entries_queue_id_order_index
	ON merge_queue_entries (merge_queue_entry_queue_id, merge_queue_entry_order_index);

CREATE UNIQUE INDEX idx_merge_queue_entries_merge_commit_sha
	ON merge_queue_entries (merge_queue_entry_merge_commit_sha)
	WHERE merge_queue_entry_merge_commit_sha IS NOT NULL;

CREATE INDEX idx_merge_queue_entries_merge_deadline
	ON merge_queue_entries (merge_queue_entry_checks_deadline)
	WHERE merge_queue_entry_state = 'checks_in_progress';
