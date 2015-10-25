# Encrypt Secrets

The `secure` command lets your create a signed and encrypted `.drone.sec` file. The `.drone.sec` file is decrypted at runtime and will inject your secrets into your `.drone.yml` file.

## Usage

Create a plaintext `.drone.sec.yml` file that contains your secret variables:

```
environment:
  - PASSWORD=foo
```

Encrypt and generate the `.drone.sec` file:

```
drone secure --repo octocat/hello-world
```

Commit the encrypted `.drone.sec` file to your repository.

Note the `drone secure` command will automatically calculate the shasum of your yaml file and store in the `.drone.sec` file. This prevent secrets from being injected into the build if the Yaml changes.


## Arguments

The `secure` command accepts the following arguments: 

* `--in` secrets in plain text yaml. defaults to `.drone.sec.yml`
* `--out` encrypted secret file. defaults to `.drone.sec`
* `--yaml` location of your `.drone.yml` file. defaults to `.drone.yml`
* `--repo` name of your repository **required**


## Shared Secrets

You cannot re-use the same `.drone.sec` for multiple repositories. You can, however, use the same plaintext secret file for multiple repositories.

```
cd octocat/hello-world
drone secure --in $HOME/.drone.sec.yml --repo octocat/hello-world

cd octocat/Spoon-Knife
drone secure --in $HOME/.drone.sec.yml --repo octocat/Spoon-Knife
```