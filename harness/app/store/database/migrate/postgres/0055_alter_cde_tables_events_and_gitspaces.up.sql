ALTER TABLE gitspace_events DROP CONSTRAINT fk_geven_gitspace_config_id;
ALTER TABLE gitspace_events DROP COLUMN geven_gitspace_config_id;
ALTER TABLE gitspace_events
    ADD COLUMN geven_entity_type TEXT NOT NULL;
ALTER TABLE gitspace_events
    ADD COLUMN geven_entity_uid TEXT;
ALTER TABLE gitspace_events
    ADD COLUMN geven_entity_id INTEGER NOT NULL;
CREATE INDEX gitspace_events_entity_id ON gitspace_events (geven_entity_id);

ALTER TABLE gitspaces DROP CONSTRAINT fk_gits_infra_provisioned_id;
ALTER TABLE gitspaces DROP COLUMN gits_infra_provisioned_id;
DROP INDEX gitspaces_gitspace_config_id_space_id;
ALTER TABLE gitspaces
    ADD COLUMN gits_uid TEXT NOT NULL;
CREATE UNIQUE INDEX gitspaces_uid_space_id ON gitspaces (gits_uid, gits_space_id);