# GitHub

Drone comes with built-in support for GitHub and GitHub Enterprise. To enable GitHub you should configure the GitHub driver using the following environment variables:

```bash
REMOTE_DRIVER=github
REMOTE_CONFIG=https://github.com?client_id=${client_id}&client_secret=${client_secret}
```

## GitHub configuration

The following is the standard URI connection scheme:

```
scheme://host[:port][?options]
```

The components of this string are:

* `scheme` server protocol `http` or `https`.
* `host` server address to connect to. The default value is github.com if not specified.
* `:port` optional. The default value is :80 if not specified.
* `?options` connection specific options.

## GitHub options

This section lists all connection options used in the connection string format. Connection options are pairs in the following form: `name=value`. The value is always case sensitive. Separate options with the ampersand (i.e. &) character:

* `client_id` oauth client id for registered application.
* `client_secret` oauth client secret for registered application.
* `open=false` allows users to self-register. Defaults to false..
* `orgs=drone&orgs=docker` restricts access to these GitHub organizations. **Optional**
* `private_mode=false` indicates GitHub Enterprise is running in private mode.
* `skip_verify=false` skip ca verification if self-signed certificate. Defaults to false.

## GitHub registration

You must register your application with GitHub in order to generate a Client and Secret. Navigate to your account settings and choose Applications from the menu, and click Register new application.

Please use `http://drone.mycompany.com/authorize` as the Authorization callback URL.
