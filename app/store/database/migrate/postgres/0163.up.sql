ALTER TABLE pullreqs
    ADD COLUMN pullreq_substate TEXT NOT NULL DEFAULT '';

CREATE TABLE auto_merges (
    auto_merge_pullreq_id    INTEGER NOT NULL,
    auto_merge_requested     BIGINT NOT NULL,
    auto_merge_requested_by  INTEGER NOT NULL,
    auto_merge_method        TEXT NOT NULL,
    auto_merge_title         TEXT NOT NULL,
    auto_merge_message       TEXT NOT NULL,
    auto_merge_delete_branch BOOLEAN NOT NULL,

    CONSTRAINT pk_auto_merge
        PRIMARY KEY (auto_merge_pullreq_id),

    CONSTRAINT fk_auto_merge_pullreq FOREIGN KEY (auto_merge_pullreq_id)
        REFERENCES pullreqs (pullreq_id) ON DELETE CASCADE,

    CONSTRAINT fk_auto_merge_created_by FOREIGN KEY (auto_merge_requested_by)
        REFERENCES principals
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);
