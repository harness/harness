CREATE TABLE delegate_provision_details
(
    dpdeta_id                   SERIAL PRIMARY KEY,
    dpdeta_task_id              TEXT    NOT NULL,
    dpdeta_action_type          TEXT    NOT NULL,
    dpdeta_gitspace_instance_id INTEGER NOT NULL,
    dpdeta_space_id             INTEGER NOT NULL,
    dpdeta_agent_port           INTEGER NOT NULL,
    dpdeta_created              BIGINT  NOT NULL,
    dpdeta_updated              BIGINT  NOT NULL,
    CONSTRAINT fk_dpdeta_gitspace_instance_id FOREIGN KEY (dpdeta_gitspace_instance_id)
        REFERENCES gitspaces (gits_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    CONSTRAINT fk_dpdeta_space_id FOREIGN KEY (dpdeta_space_id)
        REFERENCES spaces (space_id) MATCH SIMPLE
        ON UPDATE NO ACTION
);

CREATE UNIQUE INDEX delegate_provision_details_task_id_space_id ON delegate_provision_details (dpdeta_task_id, dpdeta_space_id);