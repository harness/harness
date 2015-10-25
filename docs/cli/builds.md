# Local Builds

The `exec` command lets you run a build on your personal machine (ie your laptop). It does not involve the Drone server in any way. This is very useful for local testing and troubleshooting.

## Instructions

The `drone exec` command should be executed from the root of your repository, where the `.drone.yml` file is located:

```
cd octocat/hello-world
drone exec
```

This only executes a `build` step. It does not execute `clone`, `publish`, `deploy`, or `notify` steps, nor will it decrypt your `.drone.sec` file.

## Arguments

The `exec` command accepts the following arguments: 

* `DOCKER_HOST` - docker deamon address. defaults to `unix:///var/run/docker.sock`
* `DOCKER_TLS_VERIFY` - docker daemon supports tlsverify
* `DOCKER_CERT_PATH` - docker certificate directory


## Boot2Docker

You may use the `drone exec` command with boot2docker as long as your code exists within your `$HOME` directory. This is because boot2docker mounts your home directory into the virtualbox instance giving the Docker daemon access to your local files.

## Known Issues

Attempting to cancel (`ctrl+C`) a running build will leave behind orphan containers. This is a known issue and we are planning a fix.

## Limitations

You cannot use `drone exec` with a remote Docker instance. Your local codebase is shared via a volume with the Docker daemon, which is not possible when communicating with a remote Docker host on a different machine.