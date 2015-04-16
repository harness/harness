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
nodes = [
  "unix:///var/run/docker.sock",
  "unix:///var/run/docker.sock"
]

[service]
name = "github"
base = "https://github.com"
orgs = []
open = false
private_mode = false
skip_verify = true

[service.oauth]
client = ""
secret = ""
authorize = "https://github.com/login/oauth/authorize"
access_token = "https://github.com/login/oauth/access_token"
request_token = ""
```
