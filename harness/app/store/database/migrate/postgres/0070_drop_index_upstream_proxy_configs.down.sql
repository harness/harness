alter table upstream_proxy_configs
    add constraint fk_layers_secret_identifier_and_secret_space_id
        foreign key (upstream_proxy_config_secret_identifier, upstream_proxy_config_secret_space_id)
            references secrets(secret_uid, secret_space_id)