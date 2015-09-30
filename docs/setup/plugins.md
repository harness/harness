# Plugins

Plugins are Docker containers, executed during your build process. Plugins are downloaded automatically, on-demand as they are encountered in your `.drone.yml` file.

Plugin examples include:

* `git` plugin to clone your repository
* `gh_pages` plugin to publish documentation to GitHub pages
* `slack` plugin to notify your Slack channel when a build completes
* `s3` plugin to push files or build artifacts to your S3 bucket

See the [plugin marketplace](http://addons.drone.io) for a full catalog of official plugins.

## Security

For security reasons you must whitelist plugins. The default whitelist matches the official Drone plugins -- `plugins/drone` hosted in the [Docker registry](https://registry.hub.docker.com/repos/plugins/).

You can customize your whitelist by setting the `PLUGINS` environment variable. Note that you can use globbing to whitelist all Docker images with a defined prefix:

**Example 1:** whitelist official Drone plugins

```
PLUGINS=plugins/*
```

**Example 2:** whitelist plugins for registry user `octocat`

```
PLUGINS=octocat/*
```
