DELETE
FROM gitspace_events;
DELETE
FROM SQLITE_SEQUENCE
WHERE name = 'gitspace_events';

DELETE
FROM gitspaces;
DELETE
FROM SQLITE_SEQUENCE
WHERE name = 'gitspaces';

DELETE
FROM gitspace_configs;
DELETE
FROM SQLITE_SEQUENCE
WHERE name = 'gitspace_configs';

DELETE
FROM infra_provider_resources;
DELETE
FROM SQLITE_SEQUENCE
WHERE name = 'infra_provider_resources';

DELETE
FROM infra_provider_configs;
DELETE
FROM SQLITE_SEQUENCE
WHERE name = 'infra_provider_configs';

ALTER TABLE gitspace_configs
    DROP COLUMN gconf_code_repo_ref;