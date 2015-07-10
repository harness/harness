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

Use a more verbose `.drone.yml` syntax to declare multiple `heroku` deployment steps:

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

  # this is the same as above, but uses a short-hand syntax
  # and infers the `image` name

  heroku:
    when:
      branch: stage
```
