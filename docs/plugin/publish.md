# Publishing Plugins

Plugins must be published to a Docker registry in order for Drone to download and execute at runtime. You can publish to the official registry (at [index.docker.io](https://index.docker.io)) or to a private registry.

## Plugin marketplace

Official plugins are published to [index.docker.io/u/plugins](https://index.docker.io/u/plugins/) and are listed in the [plugin marketplace](http://addons.drone.io). If you would like to submit your plugin to the marketplace, as an officially supported Drone plugin, it must meet the following criteria:

* Plugin is useful to the broader community
* Plugin is documented
* Plugin is written in Go [1]
* Plugin has manifest file
* Plugin uses Apache2 license
* Plugin uses `gliderlabs/apline` base image (unless technical limitations prohibit)

[1] Although plugins can be written in any language, official plugins must be written in Go. The core Drone team consists of primarily Go developers and we simply lack the expertise and bandwidth to support multiple stacks. This may change in the future (remember, this is still a young project) but for now this remains a requirement.


## Plugin manifest

The plugin manifest, a subsection of the `.drone.yml` file, contains important information about your plugin:

* `name` - display name of the plugin
* `desc` - brief description of the plugin
* `type` - type of plugin. Possible values are clone, publish, deploy, notify
* `image` - image repository in the Docker registry

Here is the [example manifest](https://github.com/drone-plugins/drone-slack/blob/master/.drone.yml) for the slack plugin:

```yaml
---
plugin:
  name: Slack
  desc: Sends build status notifications to your Slack channel.
  type: notify
  image: plugins/drone-slack
  labels:
    - chat
    - messaging
```

## Plugin documentation

The plugin documentation is stored in the `./DOCS.md` file in the root of the repository. This file is used to auto-generate the documentation displayed in the [plugin marketplace](http://addons.drone.io).
