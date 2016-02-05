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


## Substitution

A subset of variables may be substituted directly into the Yaml at runtime using the `$$` notation:

* `$$BUILD_NUMBER` build number for the current build
* `$$COMMIT` git sha for the current build, long format
* `$$BRANCH` git branch for the current build
* `$$BUILD_NUMBER` build number for the current build
* `$$TAG` tag name

This is useful when you need to dynamically configure your plugin based on the current build. For example, we can alter an artifact name to include the branch:

```
publish:
  s3:
    source: ./foo.tar.gz
    target: ./foo-$$BRANCH.tar.gz
```

## Operations

A subset of bash string substitution operations are emulated:

* `$$param` parameter substitution
* `$${param}` parameter substitution (same as above)
* `"$$param"` parameter substitution with escaping
* `$${param:pos}` parameter substition with substring
* `$${param:pos:len}` parameter substition with substring
* `$${param=default}` parameter substition with default
* `$${param##prefix}` parameter substition with prefix removal
* `$${param%%suffix}` parameter substition with suffix removal
* `$${param/old/new}` parameter substition with find and replace

## Caveats to Interpolation
Due to security concerns, there are certain `conditions` in the Drone CI ecosystem where the Drone will not swap variables for values. An example of this condition, and very specifically, is the `pull_request` condition:

Tracking details: https://github.com/drone/drone/issues/1250

#### Details
`.drone.yml`
```yaml
build:
  always_run:
    image: <image_name>
    commands:
      - export KEY_NAME="$$OS_KEY_NAME"
      - env | sort
  pull_requests:
    image: <image_name>
    when:
      event: pull_request
    commands:
       - export KEY_NAME="$$OS_KEY_NAME"
       - env | sort
```

`secrets.yaml`
```yaml
environment:
  - OS_KEY_NAME=BOOM_GOES_THE_DYNAMITE
```

Build log for <b>`push`</b> (labeled as: `always_runs`):
```
$ env | sort
OS_KEY_NAME=BOOM_GOES_THE_DYNAMITE
```

Build log for <b>`pull_request`</b> (labeled as: `pull_requests`):
```
$ env | sort
OS_KEY_NAME=7OS_KEY_NAME
```
