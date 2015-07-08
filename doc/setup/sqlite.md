# SQLite

Drone uses SQLite as the default database. It requires zero configuration or installation, and can easily scale to meet the needs of most teams. If you need to customize SQLite, you must specify the `DRONE_DATASTORE` environment variable with the URI configuration string. This section describes the URI format for configuring the sqlite3 driver.

The following is the standard URI connection scheme:

```
sqlite3://path
```

The components of this string are:

* `sqlite3://` required prefix to load the sqlite3 driver
* `host` local path to sqlite database. The default value is `/var/lib/drone/drone.sqlite`.

This is an example connection string:

```bash
DRONE_DATASTORE="sqlite3:///var/lib/drone/drone.sqlite"
```
