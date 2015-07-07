# GitLab

Drone comes with built-in support for GitLab version 7.7 and higher. To enable GitLab, you must specify the `DRONE_REMOTE` environment variable with the URI configuration string. This section describes the URI format for configuring the GitLab driver.

The following is the standard URI connection scheme:

```
gitlab://host[:port][?options]
```

The components of this string are:

* `gitlab://` required prefix to load the GitLab driver
* `host` server address to connect to.
* `:port` optional. The default value is `:80` if not specified.
* `?options` connection specific options

This is an example connection string:

```bash
DRONE_REMOTE="gitlab://gitlab.hooli.com?client_id=c0aaff74c060ff4a950d&client_secret=1ac1eae5ff1b490892f5546f837f306265032412"
```

## GitLab options

This section lists all connection options used in the connection string format. Connection options are pairs in the following form: `name=value`. The value is always case sensitive. Separate options with the ampersand (i.e. &) character:

* `client_id` oauth client id for registered application
* `client_secret` oauth client secret for registered application
* `open=false` allows users to self-register. Defaults to false for security reasons.
* `orgs=drone,docker` restricts access to these GitLab organizations. **Optional**
* `skip_verify=false` skip ca verification if self-signed certificate. Defaults to false for security reasons.
* `ssl=true` initiates the connection with TLS/SSL. Defaults to true.
