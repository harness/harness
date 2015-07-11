# GitHub

Drone comes with built-in support for GitHub and GitHub Enterprise. To enable and configure GitHub, you should set the following environment variables:

```
REMOTE_DRIVER="github"

GITHUB_CLIENT="c0aaff74c060ff4a950d"
GITHUB_SECRET="1ac1eae5ff1b490892f5"
```

## GitHub settings

This section lists all environment variables used to configure GitHub.

* `GITHUB_HOST` server address to connect to. The default value is `https://github.com` if not specified.
* `GITHUB_CLIENT` oauth client id for registered application
* `GITHUB_SECRET` oauth client secret for registered application
* `GITHUB_OPEN=false` allows users to self-register. Defaults to false for security reasons.
* `GITHUB_ORGS=drone,docker` restricts access to these GitHub organizations. **Optional**
* `GITHUB_PRIVATE_MODE=false` indicates GitHub Enterprise is running in private mode

## GitHub Enterprise

If you are configuring Drone with GitHub Enterprise edition, you must specify the `GITHUB_HOST` in the configuration string. Note that you may also need to set `GITHUB_PRIVATE_MODE=true` when running GitHub Entperirse in private mode.
