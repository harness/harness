DROP TABLE delegate_provision_details;

CREATE TABLE delegate_provision_details
(
    dpdeta_id                           INTEGER PRIMARY KEY AUTOINCREMENT,
    dpdeta_task_id                      TEXT    NOT NULL,
    dpdeta_action_type                  TEXT    NOT NULL,
    dpdeta_gitspace_instance_identifier TEXT    NOT NULL,
    dpdeta_space_id                     INTEGER NOT NULL,
    dpdeta_agent_port                   INTEGER NOT NULL,
    dpdeta_created                      BIGINT  NOT NULL,
    dpdeta_updated                      BIGINT  NOT NULL,
    CONSTRAINT fk_dpdeta_gitspace_instance_identifier_space_id FOREIGN KEY (dpdeta_gitspace_instance_identifier, dpdeta_space_id)
        REFERENCES gitspaces (gits_uid, gits_space_id) MATCH SIMPLE
        ON UPDATE NO ACTION,
    CONSTRAINT fk_dpdeta_space_id FOREIGN KEY (dpdeta_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
);

CREATE UNIQUE INDEX delegate_provision_details_task_id_space_id ON delegate_provision_details (dpdeta_task_id, dpdeta_space_id);