# Deploy

Drone uses the `deploy` section of the `.drone.yml` to configure deployment steps. Drone does not have any built-in deployment capabilities. This functionality is outsourced to [plugins](http://addons.drone.io). See the [plugin marketplace](http://addons.drone.io) for a list of official plugins.

An example plugin that deploys to Heroku:

```yaml
deploy:
  heroku:
    app: pied_piper
    token: f10e2821bbbea5
```

## Deploy conditions

Use the `when` attribute to limit deployments to a specific branch:

```yaml
deploy:
  heroku:
    when:
      branch: master

  # you can also do simple matching

  google_appengine:
    when:
      branch: feature/*
```

<!--
## Deploy plugins

Deployment plugins are Docker images that attach to your build environment at runtime. They are declared in the `.drone.yml` using the following format:

```
deploy:
  [step_name:]
    [image:]
    [options]
```

An example `heroku` plugin configuration:

```yaml
deploy:
  heroku:
    image: plugins/heroku
    app: pied_piper
    token: f10e2821bbbea5
```

The `image` attribute is optional. If the `image` is not specified Drone attempts to use the `step_name` as the `image` name. The below examples both produce identical output:

```yaml
  # this step specifies an image

  heroku:
    image: plubins/heroku
    app: pied_piper
    token: f10e2821bbbea5

  # and this step infers the image from the step_name

  heroku:
    app: pied_piper
    token: f10e2821bbbea5
```

The `image` attribute is useful when you need to invoke the same plugin multiple times. For example, we may want to deploy to multiple `heroku` environments depending on branch:

```yaml
  # deploy master to our production heroku environment

  heroku_prod:
    image: plugins/heroku
    when:
      branch: master

  # deploy to our staging heroku environment

  heroku_staging:
    image: plugins/heroku
    when:
      branch: stage
```
-->
