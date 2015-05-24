You may configure Drone to integrate with GitLab (version 7.9 or higher). This can be configured in the `/etc/drone/drone.toml` configuration file:

```ini
[gitlab]
url = "https://gitlab.com"
client = "c0aaff74c060ff4a950d"
secret = "1ac1eae5ff1b490892f5546f837f306265032412"
skip_verify=false
open=false
```

Or a custom installation:

```ini
[gitlab]
url = "http://gitlab.drone.io"
client = "c0aaff74c060ff4a950d"
secret = "1ac1eae5ff1b490892f5546f837f306265032412"
skip_verify=false
open=false
```

### Environment Variables

You may also configure Gitlab using environment variables. This is useful when running Drone inside Docker containers, for example.

```bash
DRONE_GITLAB_URL="https://gitlab.com"
DRONE_GITLAB_CLIENT="c0aaff74c060ff4a950d"
DRONE_GITLAB_SECRET="1ac1eae5ff1b490892f5546f837f306265032412"
```

### User Registration

User registration is closed by default and new accounts must be provisioned in the user interface. You may allow users to self-register with the following configuration flag:

```ini
[gitlab]
open = true
```

Please note this has security implications. This setting should only be enabled if you are running Drone behind a firewall.

### Self-Signed Certs

If your Gitlab installation uses a self-signed certificate you may need to instruct Drone to skip TLS verification. This is not recommended, but if you have no other choice you can include the following:

```ini
[gitlab]
skip_verify=true
```