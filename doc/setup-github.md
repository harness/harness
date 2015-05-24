You may configure Drone to integrate with GitHub or GitHub enterprise. This can be configured in the `/etc/drone/drone.toml` configuration file:

```ini
[github]
client = "c0aaff74c060ff4a950d"
secret = "1ac1eae5ff1b490892f5546f837f306265032412"
```

### Environment Variables

You may also configure GitHub using environment variables. This is useful when running Drone inside Docker containers, for example.

```bash
DRONE_GITHUB_CLIENT="c0aaff74c060ff4a950d"
DRONE_GITHUB_SECRET="1ac1eae5ff1b490892f5546f837f306265032412"
```

### Github Enterprise

You may also configure Drone to integrate with GitHub Enterprise. Note that if you are running GitHub Enterprise in private mode you should set `private_mode=true`, forcing Drone to clone public repositories with git+ssh.

```ini
[github_enterprise]
url = "https://github.drone.io"
api = "https://github.drone.io/api/v3/"
client = "c0aaff74c060ff4a950d"
secret = "1ac1eae5ff1b490892f5546f837f306265032412"
private_mode = false
```

### User Registration

User registration is closed by default and new accounts must be provisioned in the user interface. You may allow users to self-register with the following configuration flag:

```ini
[github]
open = true
```

Please note this has security implications. This setting should only be enabled if you are running Drone behind a firewall.

### Organization Whitelists

When specified, only users belonging to these organization may login to the system. Use this option to enable self-registration while still limiting access to the general public.

```ini
[github]
orgs = [ "drone", "docker" ]
```
