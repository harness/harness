SQLite is the default database driver. Drone will automatically create a database file at `/var/lib/drone/drone.sqlite` and will handle database schema setup and migration. You can further customize the database driver and configuration in the `drone.toml` file:

```ini
[database]
driver = "sqlite3"
datasource = "/var/lib/drone/drone.sqlite"
```

You may also configure the database using environment variables:

```bash
DRONE_DATABASE_DRIVER="sqlite3"
DRONE_DATABASE_DATASOURCE="/var/lib/drone/drone.sqlite"
```

### Postgres

You may alternatively configure Drone to use a Postgres database:

```ini
[database]
driver = "postgres"
datasource = "host=127.0.0.1 user=postgres dbname=drone sslmode=disable"
```

For more details about how to configure the datasource string please see the official driver documentation:

http://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING


### MySQL

Drone does not include support for MySQL at this time. We hope to include support for MySQL in the future, once the following issue is resolved:

https://github.com/go-sql-driver/mysql/issues/257