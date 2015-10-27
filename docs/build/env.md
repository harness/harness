# Variables

Drone injects the following namespaced environment variables into every build:

* `DRONE=true`
* `DRONE_REPO` - repository name for the current build
* `DRONE_BRANCH` - branch name for the current build
* `DRONE_COMMIT` - git sha for the current build
* `DRONE_DIR` - working directory for the current build
* `DRONE_BUILD_NUMBER` - build number for the current build
* `DRONE_PULL_REQUEST` - pull request number fo the current build
* `DRONE_JOB_NUMBER` - job number for the current build
* `DRONE_TAG` - tag name for the current build

Drone also injects `CI_` prefixed variables for compatibility with other systems:

* `CI=true`
* `CI_NAME=drone`
* `CI_REPO` - repository name of the current build
* `CI_BRANCH` - branch name for the current build
* `CI_COMMIT` - git sha for the current build
* `CI_BUILD_NUMBER` - build number for the current build
* `CI_PULL_REQUEST` - pull request number fo the current build
* `CI_JOB_NUMBER` - job number for the current build
* `CI_BUILD_DIR` - working directory for the current build
* `CI_BUILD_URL` - url for the current build
* `CI_TAG` - tag name for the current build


## Injecting

A subset of variables may be injected directly into the Yaml at runtime using the `$$` notation:

* `$$COMMIT` git sha for the current build, `--short` format
* `$$BRANCH` git branch for the current build
* `$$REPO` repository full name (in `owner/name` format)
* `$$TAG` tag name

This is useful when you need to dynamically configure your plugin based on the current build. For example, we can alter an artifact name to include the branch:

```
publish:
  s3:
    source: ./foo.tar.gz
    target: ./foo-$$BRANCH.tar.gz
```
