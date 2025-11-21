DROP INDEX gitspace_events_entity_id;
ALTER TABLE gitspace_events DROP COLUMN geven_entity_type;
ALTER TABLE gitspace_events DROP COLUMN geven_entity_uid;
ALTER TABLE gitspace_events DROP COLUMN geven_entity_id;
ALTER TABLE gitspace_events
    ADD COLUMN geven_gitspace_config_id INTEGER NOT NULL;
ALTER TABLE gitspace_events
    ADD CONSTRAINT fk_geven_gitspace_config_id FOREIGN KEY (geven_gitspace_config_id)
        REFERENCES gitspace_configs (gconf_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE CASCADE;

ALTER TABLE gitspaces
    ADD COLUMN gits_infra_provisioned_id INTEGER;
ALTER TABLE gitspaces
    ADD CONSTRAINT fk_gits_infra_provisioned_id FOREIGN KEY (gits_infra_provisioned_id)
        REFERENCES infra_provisioned (iprov_id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION;
DROP index gitspaces_uid_space_id;
ALTER TABLE gitspaces DROP COLUMN gits_uid;
CREATE UNIQUE INDEX gitspaces_gitspace_config_id_space_id ON gitspaces (gits_gitspace_config_id, gits_space_id);

