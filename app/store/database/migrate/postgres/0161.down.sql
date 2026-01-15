ALTER TABLE upstream_proxy_configs
    DROP COLUMN IF EXISTS upstream_proxy_config_firewall_mode;

DROP TYPE IF EXISTS upstream_proxy_config_firewall_mode;
