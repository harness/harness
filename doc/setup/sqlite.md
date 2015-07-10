# SQLite

Drone uses SQLite as the default database with zero configuration required. In order to customize the SQLite database configuration you should specify the following environment variables:

```
DATASTORE_DRIVER="sqlite3"
DATASTORE_CONFIG="/var/lib/drone/drone.sqlite"
```

## Sqlite3 connection

The components of this connection string are:

* `path` local path to sqlite database. The default value is `/var/lib/drone/drone.sqlite`.

This is an example connection string:

```
/var/lib/drone/drone.sqlite
```
