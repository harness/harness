# Overview

Drone has a robust plugin model that allows you to extend the platform to meet your unique project needs. This documentation describes the Drone plugin architecture and will help you create your own custom plugins. If you are searching for existing plugins please see the [plugin marketplace](http://addons.drone.io).

## How Plugins Work

Plugins are simply Docker containers that attach to your build at runtime and perform custom operations. These operations may include deploying code, publishing artifacts and sending notifications.

Plugins are declared in the `.drone.yml` file. When Drone encounters a plugin it attempts to download from the registry (if not already downloaded) and then execute. This is an example of the [slack plugin](https://github.com/drone-plugins/drone-slack) configuration:

```yaml
---
notify:
  slack:
    image: plugins/slack
    webhook_url: https://hooks.slack.com/services/...
    username: captain_freedom
    channel: ics
```

Plugins receive plugin configuration data, repository and build data as an encoded JSON string. The plugin can use this data when executing its task. For example, the [slack plugin](https://github.com/drone-plugins/drone-slack) uses the build and repository details to format and send a message to a channel.

Plugins also have access to the `/drone` volume, which is shared across all containers, including the build container. The repository is cloned to a subdirectory of `/drone/src`, which means plugins have access to your source code as well as any generated assets (binary files, reports, etc). For example, the [heroku plugin](https://github.com/drone-plugins/drone-heroku) accesses your source directory and executes `git push heroku master` to deploy your code.

## Plugin Input

Plugins receive build details via `arg[1]` as a JSON encoded string. The payload includes the following data structures:

* `repo` JSON representation of the repository
* `build` JSON representation of the build, including commit and pull request
* `vargs` JSON representation of the plugin configuration, as defined in the `.drone.yml`

Drone provides a simple [plugin library](https://github.com/drone/drone-plugin-go) that helps read and unmarshal the input parameters:

```go
func main() {
    var repo = plugin.Repo{}
    var build = plugin.Build{}
    var vargs = struct {
        Webhook  string `json:"webhook_url"`
        Username string `json:"username"`
        Channel  string `json:"channel"`
    }{}

    plugin.Param("repo", &repo)
    plugin.Param("build", &build)
    plugin.Param("vargs", &vargs)
    plugin.Parse()

    // send slack notification
}
```

## Plugin Output

Plugins cannot send structured data back to Drone. Plugins can, however, write information to stdout so that it appears in the logs. Plugins can fail a build by exiting with a non-zero status code.

You may be asking yourself "how do I send reports or metrics back to Drone" or "how do I generate and store artificats in Drone"? The answer is that you cannot. Instead, you should use plugins to generate reports and send to third party services (like Coveralls) or generate and upload artifacts to third party storage services (like Bintray or S3).

## Plugin Reference

These are existing plugins that you can reference when building your own:

* [Slack](https://github.com/drone-plugins/drone-slack) - publish messages to a Slack channel when your build finishes
* [Heroku](https://github.com/drone-plugins/drone-heroku) - deploy your application to Heroku
* [Docker](https://github.com/drone-plugins/drone-docker) - build and publish your Docker image to a registry
