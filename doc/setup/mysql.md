> **NOTE** the mysql driver is disable until driver issue [#257](https://github.com/go-sql-driver/mysql/issues/257) is resolved

# MySQL

Drone comes with support for MySQL as an alternate database engine. To enable MySQL, you should specify the following environment variables:

```bash
DATABASE_DRIVER="mysql"
DATABASE_CONFIG="root:pa55word@tcp(localhost:3306)/drone"
```

## MySQL configuration

The following is the standard URI connection scheme:

```
[username[:password]@][protocol[(address)]]/dbname[?options]
```

The components of this string are:

* `username` optional. Use this username when connecting to the MySQL instance.
* `password` optional. Use this password when connecting to the MySQL instance.
* `protocol` server protocol to connect with.
* `address` server address to connect to.
* `dbname` name of the database to connect to
* `?options` connection specific options

This is an example connection string:

```
root:pa55word@tcp(localhost:3306)/drone
```

## MySQL options

See the official [driver documentation](https://github.com/go-sql-driver/mysql#parameters) for a full list of driver options.
