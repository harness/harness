# SQLite

Drone uses SQLite as the default database with zero configuration required. In order to customize the SQLite database configuration you should specify the following environment variables:

```bash
DATABASE_DRIVER="sqlite3"
DATABASE_CONFIG="/var/lib/drone/drone.sqlite"
```

## Sqlite3 configuration

The components of the datasource connection string are:

* `path` local path to sqlite database. The default value is `/var/lib/drone/drone.sqlite`.

This is an example connection string:

```bash
DATABASE_CONFIG="/var/lib/drone/drone.sqlite"
```
