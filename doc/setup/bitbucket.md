> **NOTE** Bitbucket integration has not yet been merged into 0.4, but is planned in the near future

# Bitbucket

Drone comes with built-in support for Bitbucket. To enable and configure Bitbucket, you should set the following environment variables:

```
REMOTE_DRIVER="bitbucket"

BITBUCKET_KEY="c0aaff74c060ff4a950d"
BITBUCKET_SECRET="1ac1eae5ff1b490892f5"
BITBUCKET_OPEN="true"
BITBUCKET_ORGS="drone,drone-plugins"
```

## Bitbucket settings

This section lists all environment variables used to configure Bitbucket.

* `BITBUCKET_KEY` oauth client id for registered application
* `BITBUCKET_SECRET` oauth client secret for registered application
* `BITBUCKET_OPEN=false` allows users to self-register. Defaults to false for security reasons.
* `BITBUCKET_ORGS=drone,docker` restricts access to these Bitbucket organizations. **Optional**
