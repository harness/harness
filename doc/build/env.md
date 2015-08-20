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

You may also store encrypted, private variables in the `.drone.yml` and inject at runtime. Private variables are encrypted using RSA encryption with OAEP (see [EncryptOAEP](http://golang.org/pkg/crypto/rsa/#EncryptOAEP)). You can generate encrypted strings from your repository settings screen.

Once you have an ecrypted string, you can add to the `secure` section of the `.drone.yml`.These variables are decrypted and injected into the `.drone.yml` at runtime using the `$$` notation.

An example `.drone.yml` expecting the `HEROKU_TOKEN` private variable:

```yaml
build:
  image: golang
  commands:
    - go get
    - go build
    - go test

deploy:
  heroku:
    app: pied_piper
    token: $$HEROKU_TOKEN

secure:
  HEROKU_TOKEN: <encrypted string>
```
