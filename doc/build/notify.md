# Notify

Drone uses the `notify` section of the `.drone.yml` to configure notification steps. Drone does not have any built-in notification capabilities. This functionality is outsourced to [plugins](http://addons.drone.io). See the [plugin marketplace](http://addons.drone.io) for a list of official plugins.

An example configuration that sends a Slack notification on build completion:

```yaml
notify:
  slack:
    webhook_url: https://hooks.slack.com/services/f10e2821bbb/200352313bc
    channel: dev
    username: drone
```

## Notification conditions

Use the `when` attribute to limit execution to a specific branch:

```yaml
publish:
  slack:
    when:
      branch: master
```

Or limit execution based on the build status. The below example will only send the notification when the build fails:

```yaml
publish:
  slack:
    when:
      success: false
      failure: true
```
