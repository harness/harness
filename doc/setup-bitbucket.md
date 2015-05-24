You may configure Drone to integrate with Bitbucket. This can be configured in the `/etc/drone/drone.toml` configuration file:

```ini
[bitbucket]
client = "c0aaff74c060ff4a950d"
secret = "1ac1eae5ff1b490892f5546f837f306265032412"
open = false
```

Please note this has security implications. This setting should only be enabled if you are running Drone behind a firewall.

### Environment Variables

You may also configure Bitbucket using environment variables. This is useful when running Drone inside Docker containers, for example.

```bash
DRONE_BITBUCKET_CLIENT="c0aaff74c060ff4a950d"
DRONE_BITBUCKET_SECRET="1ac1eae5ff1b490892f5546f837f306265032412"
```

### User Registration

User registration is closed by default and new accounts must be provisioned in the user interface. You may allow users to self-register with the following configuration flag:

```ini
[bitbucket]
open = true
```