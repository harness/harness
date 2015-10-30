# Gitlab

> GitLab support is experimental and unstable due to high variation in GitLab configurations and versions. We highly recommend using [Gogs](https://github.com/gogits/gogs) as an alternative to GitLab.

Drone comes with built-in support for GitLab version 8.0 and higher. To enable Gitlab you should configure the Gitlab driver using the following environment variables:

```bash
REMOTE_DRIVER=gitlab
REMOTE_CONFIG=https://gitlab.hooli.com?client_id=${client_id}&client_secret=${client_secret}
```

## Gitlab configuration

The following is the standard URI connection scheme:

```
scheme://host[:port][?options]
```

The components of this string are:

* `scheme` server protocol `http` or `https`.
* `host` server address to connect to. The default value is github.com if not specified.
* `:port` optional. The default value is :80 if not specified.
* `?options` connection specific options.

## GitLab options

This section lists all connection options used in the connection string format. Connection options are pairs in the following form: `name=value`. The value is always case sensitive. Separate options with the ampersand (i.e. &) character:

* `client_id` oauth client id for registered application
* `client_secret` oauth client secret for registered application
* `open=false` allows users to self-register. Defaults to false for security reasons.
* `orgs=drone&orgs=docker` restricts access to these GitLab organizations. **Optional**
* `skip_verify=false` skip ca verification if self-signed certificate. Defaults to false for security reasons.
* `clone_mode=token` a strategy for clone authorization, by default use repo token, but can be changed to `oauth` ( This is not secure, because your user token, with full access to your gitlab account will be written to .netrc, and it can be read by all who have access to project builds )

## Gitlab registration

You must register your application with GitLab in order to generate a Client and Secret. Navigate to your account settings and choose Applications from the menu, and click New Application.

Please use `http://drone.mycompany.com/authorize` as the Authorization callback URL.
