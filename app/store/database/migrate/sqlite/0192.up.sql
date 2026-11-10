CREATE TABLE pullreq_watchers (
 pullreq_watcher_pullreq_id INTEGER NOT NULL
,pullreq_watcher_principal_id INTEGER NOT NULL
,pullreq_watcher_created BIGINT NOT NULL
,CONSTRAINT pk_pullreq_watchers PRIMARY KEY (pullreq_watcher_pullreq_id, pullreq_watcher_principal_id)
,CONSTRAINT fk_pullreq_watcher_pullreq_id FOREIGN KEY (pullreq_watcher_pullreq_id)
    REFERENCES pullreqs (pullreq_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
,CONSTRAINT fk_pullreq_watcher_principal_id FOREIGN KEY (pullreq_watcher_principal_id)
    REFERENCES principals (principal_id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE CASCADE
);
