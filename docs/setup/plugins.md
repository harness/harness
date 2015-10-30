# Plugins

Plugins are Docker containers, executed during your build process. Plugins are downloaded automatically, on-demand as they are encountered in your `.drone.yml` file.

See the [plugin marketplace](http://addons.drone.io) for a full catalog of official plugins.

## Security

For security reasons you must whitelist plugins. The default whitelist includes the official Drone plugins hosted in the [Docker registry](https://registry.hub.docker.com/repos/plugins/). Customize your whitelist by setting the `PLUGIN_FILTER` environment variable. This is a space-separated list and includes glob matching capabilities.

Whitelist official Drone plugins

```
PLUGIN_FILTER=plugins/*
```

Whitelist official Drone plugins and registry user `octocat`

```
PLUGIN_FILTER=plugins/* octocat/*
```
