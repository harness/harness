# Bitbucket

Drone comes with built-in support for Bitbucket. To enable Bitbucket you should configure the Bitbucket driver using the following environment variables:

```bash
REMOTE_DRIVER=bitbucket
REMOTE_CONFIG=https://bitbucket.org?client_id=${client_id}&client_secret=${client_secret}
```

## Bitbucket configuration

The following is the standard URI connection scheme:

```
scheme://host[?options]
```

The components of this string are:

* `scheme` server protocol `http` or `https`.
* `host` server address to connect to.
* `?options` connection specific options.

## Bitbucket options

This section lists all connection options used in the connection string format. Connection options are pairs in the following form: `name=value`. The value is always case sensitive. Separate options with the ampersand (i.e. &) character:

* `client_id` oauth client id for registered application.
* `client_secret` oauth client secret for registered application.
* `open=false` allows users to self-register. Defaults to false.
* `orgs=drone&orgs=docker` restricts access to these Bitbucket organizations. **Optional**

## Bitbucket registration

You must register your application with Bitbucket in order to generate a Client and Secret. Navigate to your account settings and choose OAuth from the menu, and click Add Consumer.

Please use `http://drone.mycompany.com/authorize` as the Authorization callback URL. You will also need to check the following permissions:

* Account:Email
* Account:Read
* Team Membership:Read
* Repositories:Read
* Webhooks:Read and Write

## Known Issues

This section details known issues and planned features:

* Pull Request support
* Mercurial support
