UPDATE upstream_proxy_configs
SET upstream_proxy_config_auth_type = 'Anonymous'
WHERE upstream_proxy_config_auth_type = '';
