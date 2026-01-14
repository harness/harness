ALTER TABLE infra_provisioned
    DROP CONSTRAINT fk_iprov_gitspace_id;

DROP TABLE gitspace_events;
DROP TABLE gitspaces;
DROP TABLE infra_provisioned;
DROP TABLE gitspace_configs;
DROP TABLE infra_provider_resources;
DROP TABLE infra_provider_templates;
DROP TABLE infra_provider_configs;