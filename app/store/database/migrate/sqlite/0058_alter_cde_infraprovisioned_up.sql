ALTER TABLE infra_provisioned ADD COLUMN iprov_proxy_host TEXT DEFAULT '';
ALTER TABLE infra_provisioned ADD COLUMN iprov_proxy_port INTEGER DEFAULT 0;
