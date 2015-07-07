# GitHub

Drone comes with built-in support for GitHub and GitHub Enterprise. To enable GitHub, you must specify the `DRONE_REMOTE` environment variable with the URI configuration string. This section describes the URI format for configuring the github driver.

The following is the standard URI connection scheme:

```
github://host[:port][?options]
```

The components of this string are:

* `github://` required prefix to load the github driver
* `host` server address to connect to. The default value is `github.com` if not specified.
* `:port` optional. The default value is `:80` if not specified.
* `?options` connection specific options

This is an example connection string:

```bash
DRONE_REMOTE="github://github.com?client_id=c0aaff74c060ff4a950d&client_secret=1ac1eae5ff1b490892f5546f837f306265032412"
```

## GitHub options

This section lists all connection options used in the connection string format. Connection options are pairs in the following form: `name=value`. The value is always case sensitive. Separate options with the ampersand (i.e. &) character:

* `client_id` oauth client id for registered application
* `client_secret` oauth client secret for registered application
* `open=false` allows users to self-register. Defaults to false for security reasons.
* `orgs=drone,docker` restricts access to these GitHub organizations. **Optional**
* `private_mode=false` indicates GitHub Enterprise is running in private mode
* `ssl=true` initiates the connection with TLS/SSL. Defaults to true.

## GitHub Enterprise

If you are configuring Drone with GitHub Enterprise edition, you must specify the `host` in the configuration string. Note that you may also need to set `private_mode=true` when running GitHub Entperirse in private mode.
