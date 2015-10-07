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

Drone prefers `git+https` for authentication because it allows you to clone multiple private repositories. This is helpful when you have git submodules or third party dependencies you need to download (via `go get` or `npm install` or others) that are sourced from a private repository.

Drone only injects the `netrc` and `id_rsa` files into your build environment if your repository is private, or running in private mode. We do this for security reasons to avoid leaking sensitive data.
