Highly experimental branch that implements the following features:

* Pluggable database backends
* Pluggable queue
* Matrix builds
* Build plugins
* New Yaml syntax
* and more ...

Running Drone:

```
./drone --config="/path/to/config.toml"
```

Configuring Drone:

```toml
[server]
addr = ":80"
cert = ""
key = ""

[session]
secret = ""
expires = ""

[database]
path = "/etc/drone/drone.db"

[docker]
cert = ""
key = ""
addr = "unix:///var/run/docker.sock"
swarm = ""

[service]
kind = "github"
base = "https://github.com"
orgs = []
open = false
private = false
skip_verify = true

[auth]
client = ""
secret = ""
authorize = "https://github.com/login/oauth/authorize"
access_token = "https://github.com/login/oauth/access_token"
request_token = ""

[agents]
secret = ""
```

Configuration settings can also be set by environment variables using the scheme `DRONE_<section>_<confkey>`, substituting the section title for `<section>` and the key for `<confkey>`, in all caps. For example:

```shell
#!/bin/bash
# prepare environment for executing drone
DRONE_DOCKER_ADDR="tcp://10.0.0.1:2375"     # for [docker] section, 'addr' setting
DRONE_AUTH_CLIENT="0123456789abcdef0123AA"  # for [auth] section, 'client' setting
DRONE_AUTH_SECRET="<sha-1 hash secret>"     # for [auth] section, 'secret' setting

exec ./drone -config=drone.toml
```

_NOTE: Configuration settings from environment variables override values set in the TOML file._
