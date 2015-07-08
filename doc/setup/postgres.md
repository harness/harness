
# Postgres

Drone comes with support for Postgres as an alternate database engine. To enable Postgres, you must specify the `DRONE_DATASTORE` environment variable with the URI configuration string. This section describes the URI format for configuring the Postgres driver.

The following is the standard URI connection scheme:

```
postgres://[username:password@]host[:port]/[dbname][?options]
```

The components of this string are:

* `postgres://` required prefix to load the Postgres driver
* `username:password@` optional. Use these credentials when connecting to the Postgres instance.
* `host` server address to connect to. It may be a hostname, IP address, or UNIX domain socket.
* `:port` optional. The default value is `:5432` if not specified.
* `dbname` name of the database to connect to
* `?options` connection specific options

This is an example connection string:

```bash
DRONE_DATASTORE="postgres://root:pa55word@127.0.0.1:5432/postgres"
```

## Postgres options

This section lists all connection options used in the connection string format. Connection options are pairs in the following form: `name=value`. The value is always case sensitive. Separate options with the ampersand (i.e. &) character:

* `sslmode` initiates the connection with TLS/SSL (disable, require, verify-ca, verify-full)
* `connect_timeout` maximum wait for connection, in seconds.
* `sslcert` cert file location. The file must contain PEM encoded data.
* `sslkey` key file location. The file must contain PEM encoded data.
* `sslrootcert` location of the root certificate file. The file must contain PEM encoded data.
