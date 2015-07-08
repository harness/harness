# Clone

Drone automatically clones your repository and submodules at the start of your build. No configuration is required. You can, however, use the `clone` section of the `.drone.yml` to customize this behavior as needed.

## Clone Options

The custom clone options are:

* `depth` - creates a shallow clone with truncated history
* `recursive` - recursively clones git submodules
* `path` - relative path inside `/drone/src` where the repository is cloned

This is an example yaml configuration:

```yaml
clone:
  depth: 50
  recursive: true
  path: github.com/drone/drone
```

Which results in the following command:

```
git clone --depth=50 --recusive=true \
    https://github.com/drone/drone.git \
    /drone/src/github.com/drone/drone
```

## Clone Private Repos

Cloning a private repository requires authentication to the remote system. Drone prefers `git+https` and `netrc` to authenticate, but will fallback to `git+ssh` and deploy keys if not supported.

Drone prefers `git+ssh` for authentication because it allows you to clone multiple private repositories. This is helpful when you have git submodules or third party dependencies you need to download (via `go get` or `npm install` or others) that are sourced from a private repository.

Drone only injects the `netrc` and `id_rsa` files into your build environment if your repository is private, or running in private mode. We do this for security reasons to avoid leaking sensitive data.

## Clone Plugins

You can override the default `git` plugin by specifying an alternative plugin image. An example use case may be integrating with alternate version control systems, such as mercurial:

```yaml
clone:
  image: bradrydzewski/hg

  # below are plugin-specific parameters
  path: override/default/clone/path
  insecure: false
  verbose: true
```

Please reference the official `git` plugin and use this as a starting point for custom plugin development:
https://github.com/drone-plugins/drone-git
