# GitLab

Drone comes with built-in support for GitLab 7.7 and higher. To enable and configure GitLab, you should set the following environment variables:

```
REMOTE_DRIVER="gitlab"

GITLAB_HOST="https://gitlab.hooli.com"
GITLAB_CLIENT="c0aaff74c060ff4a950d"
GITLAB_SECRET="1ac1eae5ff1b490892f5"
GITLAB_OPEN="true"
GITLAB_ORGS="drone,drone-plugins"
GITLAB_SKIP_VERIFY="false"
```

## GitLab settings

This section lists all environment variables options used to configure GitLab.

* `GITLAB_HOST` server address to connect to.
* `GITLAB_CLIENT` oauth client id for registered application
* `GITLAB_SECRET` oauth client secret for registered application
* `GITLAB_OPEN=false` allows users to self-register. Defaults to false for security reasons.
* `GITLAB_ORGS=drone,docker` restricts access to these GitLab organizations. **Optional**
* `GITLAB_SKIP_VERIFY=false` skip certificate chain and host name. Defaults to false for security reasons.
