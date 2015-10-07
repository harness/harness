# Caching

> This feature is still considered experimental

Drone allows you to cache directories within the build workspace. When a build successfully completes, the named directories are gzipped and stored on the host machine. When a new build starts, the named directories are restored from the gzipped files. This can be used to improve the performance of your builds.

Below is an example `.drone.yml` configured to cache the `.git` and the `node_modules` directory:

```yaml
cache:
  mount:
    - node_modules
    - .git
```

## Branches and Matrix

Drone keeps a separate cache for each Branch and Matrix combination. Let's say, for example, you are using matrix builds to test `node 0.11.x` and `node 0.12.x` and you are caching `node_modules`. Drone will separately cache `node_modules` for each version of node.

## Pull Requests

Pull requests have read-only access to the cache. This means pull requests are not permitted to re-build the cache. This is done for security and stability purposes, to prevent a pull request from corrupting your cache.

## Deleting the Cache

There is currently no mechanism to automatically delete or flush the cache. This must be done manually, on each worker node. The cache is located in `/var/lib/drone/cache/`.

## Distributed Cache

This is outside the scope of Drone. You may, for example, use a distributed filesystem such as `ceph` or `gluster` mounted to `/var/lib/drone/cache/` to share the cache across nodes.
