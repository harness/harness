# Publish

Drone uses the `publish` section in `.drone.yml` to configure publish steps.
Publish steps are primarily used to do two things:

1. Build a Docker image from a `Dockerfile`.
2. Publish said Docker image to an image repository.

## Publishing and plugins

Drone does not have any built-in publish or artifact capabilities. 
This functionality is outsourced to [plugins](http://addons.drone.io).
Consequently, the behaviors of the various plugins can vary.

See the [plugin marketplace](http://addons.drone.io) for a list of official 
plugins and their documentation.

## How the publish step interacts with the clone/build steps

At the start of each build, Drone clones your repository in the
[clone](clone.html) step.

Once we have an up-to-date clone of your repository, the [build](build.html)
step starts. A build container is fired up (based on the image you specify
in the build step) and the cloned source is mounted as a volume. The end
result is a subdirectory beneath `/drone` in your build container. From here, 
you can run tests, compile binaries, pack single page apps, or do any other 
prep work before creating a Docker image in the next step.

Once the build step concludes, we move on to the publish step. While behavior
can vary based on the plugin you choose, this is roughly equivalent to running
`docker build` and `docker push` from within your build sub-directory within
`/drone`. Your Dockerfile will typically COPY/ADD files from your build
directory.

The publish step is for building and pushing Docker images, optionally
pulling in files from the [clone](clone.html) and [build](build.html) steps.

## Basic examples

The following example configuration uses the 
[Docker publish plugin](http://addons.drone.io/docker/) to build and push a 
Docker image to an image registry (Docker Hub by default):

```yaml
publish:
  docker:
    username: kevinbacon
    password: someday
    email: kevin.bacon@mail.com
    repo: foo/bar
    tag: latest
    file: Dockerfile
```

With the inclusion of the `registry` key, we can publish to other image 
registries:

```yaml
publish:
  docker:
    registry: my-internal-registry.io
    username: charliesheen
    password: winning
    email: charlie.sheen@mail.com
    repo: foo/bar
    tag: latest
    file: Dockerfile
```

We can also elect to use Dockerfiles named something other than `Dockerfile`:

```yaml
publish:
  docker:
    username: garybusey
    password: southernfriedchicken
    email: gary.busey@mail.com
    repo: foo/bar
    tag: latest
    file: Dockerfile.prod
```

See the [Docker publish plugin documentation](http://addons.drone.io/docker/) 
for a full list of possible key/values.

While the Docker plugin will cover most usage cases, some registries require 
special configuration to publish to. The 
[Google Container Registry (GCR) plugin](http://addons.drone.io/google_container_registry/) 
is a good example:

```yaml
publish:
  gcr:
    repo: foo/bar
    token: >
      {
        "private_key_id": "...",
        "private_key": "...",
        "client_email": "...",
        "client_id": "...",
        "type": "..."
      }
```

In this case, we end up passing in Google Cloud credentials instead of the 
things you'd typically see in a `docker push`.

## Publish conditions

Use the `when` attribute to limit execution to a specific branch:

```yaml
publish:
  docker:
    # <docker publish plugin key/values here>
    when:
      branch: master
```

You can also do simple matching:

```yaml
publish:
  gcr:
    # <gcr-specific key/values here>
    when:
      branch: feature/*
```
      
Or only publish when a tag is pushed:

```yaml
publish:
  docker:
    # <docker publish plugin key/values here>
    when:
      event: tag
```
