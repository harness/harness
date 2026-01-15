CREATE TYPE upstream_proxy_config_firewall_mode AS ENUM ('ALLOW', 'WARN', 'BLOCK');

ALTER TABLE upstream_proxy_configs
    ADD COLUMN upstream_proxy_config_firewall_mode upstream_proxy_config_firewall_mode DEFAULT 'ALLOW';

