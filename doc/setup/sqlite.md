# SQLite

Drone uses SQLite as the default database with zero configuration required. In order to customize the SQLite database configuration you should specify the following environment variables:

```bash
DATABASE_DRIVER="sqlite3"
DATABASE_CONFIG="/var/lib/drone/drone.sqlite"
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

This section lists all connection options used in the connection string format. Connection options are pairs in the following form: `name=value`. The value is always case sensitive. Separate options with the ampersand (i.e. &) character:

* `vfs` opens the database connection using the VFS value.
* `mode` opens the database as `ro`, `rw`, `rwc` or `memory`.
* `cache` opens the database with `shared` or `private` cache.
* `psow` overrides the powersafe overwrite property of the database file being opened.
* `_loc` sets the location of the time format. Use `auto` to auto-detect.
* `_busy_timeout` sets the value of the `sqlite3_busy_timeout`
* `_txlock` sets the locking behavior to `immediate`, `deferred`, or `exclusive`
