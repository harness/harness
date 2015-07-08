# Variables

The build environment has access to the following environment variables:

* `CI=true`
* `DRONE=true`
* `DRONE_REPO` - repository name for the current build
* `DRONE_BUILD` - build number for the current build
* `DRONE_BRANCH` - branch name for the current build
* `DRONE_COMMIT` - git sha for the current build
* `DRONE_DIR` - working directory for the current build


## Private Variables

Drone also lets you to store sensitive data external to the `.drone.yml` and inject at runtime. You can declare private variables in the repository settings screen. These variables are injected into the `.drone.yml` at runtime using the `$$` notation.

An example `.drone.yml` expecting the `HEROKU_TOKEN` private variable:

```yaml
build:
  image: golang
  commands:
    - go get
    - go build
    - go test

publish:
  heroku:
    app: pied_piper
    token: $$HEROKU_TOKEN
```
