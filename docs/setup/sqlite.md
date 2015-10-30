# SQLite

Drone uses SQLite as the default database with zero configuration required. In order to customize the SQLite database configuration you should specify the following environment variables:

```bash
DATABASE_DRIVER=sqlite3
DATABASE_CONFIG=/var/lib/drone/drone.sqlite
```

## Sqlite3 configuration

The following is the standard URI connection scheme:

```
file:path[?options]
```

The components of the datasource connection string are:

* `file:` URI prefix to identify database files.
* `path` local path to the database file. The default value is `/var/lib/drone/drone.sqlite`.
* `?options` connection specific options. **not recommended**

## Sqlite3 options

See the official [driver documentation](https://www.sqlite.org/uri.html#coreqp) for a full list of driver options.
