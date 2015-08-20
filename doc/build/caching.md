# Caching
Drone allows the caching of directories within the workspace. This can be used to improve the performance of the builds. This document discusses how caching is implemented and provides some general advice for lowering the execution time of builds.

Remember that cached volumes are retained between builds and those volumes are accessible through plugins. They are also available through the host's file system. Any information that is sensitive should not be contained within the cache.

## Caching Implementation
Drone uses Docker volumes to handle caching. The volumes are mounted within the build container and are available immediately within the container and any plugins running within the environment.

Given the following configuration.
```yaml
build:
  cache:
    - foo
    - fizz/buzz
```
Two volumes will be created with their paths resembling the following, with the path left of the : being the location on the host and to the right being the directory within the container. If a different path is specified in the [clone](clone.md) plugin then the paths on the right will be relative to the value in `path` rather than the default paths specified below.
```
/tmp/drone/cache/${origin}/${org}/${name}/foo:/drone/src/${origin}/${org}/${name}/foo
/tmp/drone/cache/${origin}/${org}/${name}/fizz/buzz:/drone/src/${origin}/${org}/${name}/fizz/buzz
```
If the drone repository used this caching then the volumes would look like.
```
/tmp/drone/cache/github.com/drone/drone/foo:/drone/src/github.com/drone/drone/foo
/tmp/drone/cache/github.com/drone/drone/fizz/buzz:/drone/src/github.com/drone/drone/fizz/buzz
```

### Sharing the Cache
Volumes in Docker are local to the host. When using multiple build machines there will be a copy of the cache on each of these machines. If its desireable to share the cached directories among all the build machines then a distributed file system could be mounted to `/tmp/drone/cache`. Setting up a distrubted file system is beyond the scope of this document but can potentially benifit build times and reduce the disk usage from caching.

### Deleting the Cache
The caching implementation uses the `/tmp/` directory which is typically cleared every boot unless explicitly changed. If the docker host is never rebooted then a cron job could be used to clear the `/tmp/drone/cache/` periodically. The amount of time to wait clear is dependent upon the number of builds being run, the amount of data being cached, and the size of the repositories. Because of this it is recommended to monitor the disk usage and then make a judgement call based on that data.

## Optimization Strategies
The following sections, arranged alphabetically, contain optimization strategies for various programming languages and version control systems using caching.
### C/C++
_Looking for additional input_
### Dart
_Looking for additional input_
### Git
When git is used as the version control system the .git directory is a good candidate for caching. Since the git plugin handles the checkout process the only thing needed is to add the .git directory to the cache section.
```yaml
build:
  cache:
    - .git
```
### Golang
_Looking for additional input_
### Java
_Looking for additional input_
### JavaScript
_Looking for additional input_
### PHP
_Looking for additional input_
### Python
_Looking for additional input_
### Ruby
_Looking for additional input_
