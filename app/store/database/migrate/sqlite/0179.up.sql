CREATE TABLE pullreq_reviewer_suggestions (
    pullreq_reviewer_suggestion_pullreq_id INTEGER NOT NULL,
    pullreq_reviewer_suggestion_created_by INTEGER NOT NULL,
    pullreq_reviewer_suggestion_principal_id INTEGER NOT NULL,
    pullreq_reviewer_suggestion_created BIGINT NOT NULL,

    PRIMARY KEY (pullreq_reviewer_suggestion_pullreq_id, pullreq_reviewer_suggestion_principal_id),

    CONSTRAINT fk_pullreq_reviewer_suggestions_pullreq_id FOREIGN KEY (pullreq_reviewer_suggestion_pullreq_id)
        REFERENCES pullreqs (pullreq_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_pullreq_reviewer_suggestions_created_by FOREIGN KEY (pullreq_reviewer_suggestion_created_by)
        REFERENCES principals (principal_id),
    CONSTRAINT fk_pullreq_reviewer_suggestions_principal_id FOREIGN KEY (pullreq_reviewer_suggestion_principal_id)
        REFERENCES principals (principal_id)
);