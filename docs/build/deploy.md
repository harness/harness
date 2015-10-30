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

Declare multiple `heroku` deployment steps:

```yaml
  # deploy master to our production heroku environment

  heroku:
    app: app.com
    when:
      branch: master

  # deploy to our staging heroku environment

  heroku:
    app: staging.app.com
    when:
      branch: stage
```
