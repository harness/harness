> mysql driver has known timeout issues. See [#257](https://github.com/go-sql-driver/mysql/issues/257).

# MySQL

Drone comes with support for MySQL as an alternate database engine. To enable MySQL, you should specify the following environment variables:

```bash
DATABASE_DRIVER="mysql"
DATABASE_CONFIG="root:pa55word@tcp(localhost:3306)/drone?parseTime=true"
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
root:pa55word@tcp(localhost:3306)/drone?parseTime=true
```

Please note that `parseTime` is a **required** paramter.

## MySQL options

See the official [driver documentation](https://github.com/go-sql-driver/mysql#parameters) for a full list of driver options. Note that the `parseTime=true` is required.


## MySQL Database

Drone does not automatically create the database. You should use the command line utility or your preferred management console to create the database:

```bash
mysql -P 3306 --protocol=tcp -u root -e 'create database if not exists drone;'
```
