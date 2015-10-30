# Secret Variables

> this feature is still considered experimental

Drone allows you to store secret variables in an encrypted `.drone.sec` file in the root of your repository. This is useful when your build requires sensitive information that should not be stored in plaintext in your `.drone.yml` file.

An example `.drone.sec` yaml file, prior to being encryped:

```yaml
checksum: f63561783e550ccd21663d13eaf6a4d252d84147
environment:
  - HEROKU_TOKEN=pa$$word
```

To encrypt the above yaml file

* navigate to your repository settings
* click the section labeled secret variables
* enter the plaintext yaml string in the textarea
* click the encrypt button

An encrypted string is returned to the browser. This string should be copied and pasted into a `.drone.sec` file in the root of your repository, alongside your `.drone.yml` file.

## Environment

The `environment` section of the `.drone.sec` file is a list of secret variables that get injected into your `.drone.yml` file at runtime using the `$$` notation. Secret variables are not injected as environment variables. Instead, we use a simple find and replace algorithm.

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
```

## Pull Requests

Secret variables are **not** injected into to the build section of the `.drone.yml` if your repository is **public** and the build is a **pull request**. This is for security purposes to prevent a malicious pull request from leaking your secrets.

Please note that you may still want secrets available to plugins when building a pull request. This is possible if you include a checksum of the `.drone.yml` file in your `.drone.sec` file.

## Checksum

The `checksum` field in the `.drone.yml` is a sha of your `.drone.yml` file. It is optional, but highly recommended. The `checksum` is used to verify the integrity of your `.drone.yml` file. If the checksum does not match, secret variables are not injected into your Yaml.

Generate a checksum on OSX or Linux:

```
$ shasum -a 256 .drone.yml
f63561783e550ccd21663d13eaf6a4d252d84147  .drone.yml
```

Generate a checksum on Windows with powershell:

```
$ Get-FileHash .\.drone.yml -Algorithm SHA256
```
