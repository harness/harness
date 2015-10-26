# Publish

Drone uses the `publish` section of the `.drone.yml` to configure publish steps. Drone does not have any built-in publish or artifact capabilities. This functionality is outsourced to [plugins](http://addons.drone.io). See the [plugin marketplace](http://addons.drone.io) for a list of official plugins.

An example configuration that builds a Docker image and publishes to the registry:

```yaml
publish:
  docker:
    username: kevinbacon
    password: pa55word
    email: kevin.bacon@mail.com
    repo: foo/bar
    tag: latest
    file: Dockerfile
```

## Publish conditions

Use the `when` attribute to limit execution to a specific branch:

```yaml
publish:
  docker:
    when:
      branch: master

  # you can also do simple matching

  bintray:
    when:
      branch: feature/*
```
