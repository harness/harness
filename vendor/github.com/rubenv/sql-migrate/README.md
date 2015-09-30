# sql-migrate

> SQL Schema migration tool for [Go](http://golang.org/). Based on [gorp](https://github.com/go-gorp/gorp) and [goose](https://bitbucket.org/liamstask/goose).

[![Build Status](https://travis-ci.org/rubenv/sql-migrate.svg?branch=master)](https://travis-ci.org/rubenv/sql-migrate) [![GoDoc](https://godoc.org/github.com/rubenv/sql-migrate?status.png)](https://godoc.org/github.com/rubenv/sql-migrate)

Using [modl](https://github.com/jmoiron/modl)? Check out [modl-migrate](https://github.com/rubenv/modl-migrate).

## Features

* Usable as a CLI tool or as a library
* Supports SQLite, PostgreSQL, MySQL, MSSQL and Oracle databases (through [gorp](https://github.com/go-gorp/gorp))
* Can embed migrations into your application
* Migrations are defined with SQL for full flexibility
* Atomic migrations
* Up/down migrations to allow rollback
* Supports multiple database types in one project

## Installation

To install the library and command line program, use the following:

```bash
go get github.com/rubenv/sql-migrate/...
```

## Usage
### As a standalone tool
```
$ sql-migrate --help
usage: sql-migrate [--version] [--help] <command> [<args>]

Available commands are:
    down      Undo a database migration
    redo      Reapply the last migration
    status    Show migration status
    up        Migrates the database to the most recent version available
```

Each command requires a configuration file (which defaults to `dbconfig.yml`, but can be specified with the `-config` flag). This config file should specify one or more environments:

```yml
development:
    dialect: sqlite3
    datasource: test.db
    dir: migrations/sqlite3

production:
    dialect: postgres
    datasource: dbname=myapp sslmode=disable
    dir: migrations/postgres
    table: migrations
```

The `table` setting is optional and will default to `gorp_migrations`.

The environment that will be used can be specified with the `-env` flag (defaults to `development`).

Use the `--help` flag in combination with any of the commands to get an overview of its usage:

```
$ sql-migrate up --help
Usage: sql-migrate up [options] ...

  Migrates the database to the most recent version available.

Options:

  -config=config.yml   Configuration file to use.
  -env="development"   Environment.
  -limit=0             Limit the number of migrations (0 = unlimited).
  -dryrun              Don't apply migrations, just print them.
```

The `up` command applies all available migrations. By contrast, `down` will only apply one migration by default. This behavior can be changed for both by using the `-limit` parameter.

The `redo` command will unapply the last migration and reapply it. This is useful during development, when you're writing migrations.

Use the `status` command to see the state of the applied migrations:

```bash
$ sql-migrate status
+---------------+-----------------------------------------+
|   MIGRATION   |                 APPLIED                 |
+---------------+-----------------------------------------+
| 1_initial.sql | 2014-09-13 08:19:06.788354925 +0000 UTC |
| 2_record.sql  | no                                      |
+---------------+-----------------------------------------+
```

### As a library
Import sql-migrate into your application:

```go
import "github.com/rubenv/sql-migrate"
```

Set up a source of migrations, this can be from memory, from a set of files or from bindata (more on that later):

```go
// Hardcoded strings in memory:
migrations := &migrate.MemoryMigrationSource{
    Migrations: []*migrate.Migration{
        &migrate.Migration{
            Id:   "123",
            Up:   []string{"CREATE TABLE people (id int)"},
            Down: []string{"DROP TABLE people"},
        },
    },
}

// OR: Read migrations from a folder:
migrations := &migrate.FileMigrationSource{
    Dir: "db/migrations",
}

// OR: Use migrations from bindata:
migrations := &migrate.AssetMigrationSource{
    Asset:    Asset,
    AssetDir: AssetDir,
    Dir:      "migrations",
}
```

Then use the `Exec` function to upgrade your database:

```go
db, err := sql.Open("sqlite3", filename)
if err != nil {
    // Handle errors!
}

n, err := migrate.Exec(db, "sqlite3", migrations, migrate.Up)
if err != nil {
    // Handle errors!
}
fmt.Printf("Applied %d migrations!\n", n)
```

Note that `n` can be greater than `0` even if there is an error: any migration that succeeded will remain applied even if a later one fails.

Check [the GoDoc reference](https://godoc.org/github.com/rubenv/sql-migrate) for the full documentation.

## Writing migrations
Migrations are defined in SQL files, which contain a set of SQL statements. Special comments are used to distinguish up and down migrations.

```sql
-- +migrate Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE people (id int);


-- +migrate Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE people;
```

You can put multiple statements in each block, as long as you end them with a semicolon (`;`).

If you have complex statements which contain semicolons, use `StatementBegin` and `StatementEnd` to indicate boundaries:

```sql
-- +migrate Up
CREATE TABLE people (id int);

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION do_something()
returns void AS $$
DECLARE
  create_query text;
BEGIN
  -- Do something here
END;
$$
language plpgsql;
-- +migrate StatementEnd

-- +migrate Down
DROP FUNCTION do_something();
DROP TABLE people;
```

The order in which migrations are applied is defined through the filename: sql-migrate will sort migrations based on their name. It's recommended to use an increasing version number or a timestamp as the first part of the filename.

## Embedding migrations with [bindata](https://github.com/jteeuwen/go-bindata)
If you like your Go applications self-contained (that is: a single binary): use [bindata](https://github.com/jteeuwen/go-bindata) to embed the migration files.

Just write your migration files as usual, as a set of SQL files in a folder.

Then use bindata to generate a `.go` file with the migrations embedded:

```bash
go-bindata -pkg myapp -o bindata.go db/migrations/
```

The resulting `bindata.go` file will contain your migrations. Remember to regenerate your `bindata.go` file whenever you add/modify a migration (`go generate` will help here, once it arrives).

Use the `AssetMigrationSource` in your application to find the migrations:

```go
migrations := &migrate.AssetMigrationSource{
    Asset:    Asset,
    AssetDir: AssetDir,
    Dir:      "db/migrations",
}
```

Both `Asset` and `AssetDir` are functions provided by bindata.

Then proceed as usual.

## Extending
Adding a new migration source means implementing `MigrationSource`.

```go
type MigrationSource interface {
    FindMigrations() ([]*Migration, error)
}
```

The resulting slice of migrations will be executed in the given order, so it should usually be sorted by the `Id` field.

## License 

    (The MIT License)

    Copyright (C) 2014-2015 by Ruben Vermeersch <ruben@rocketeer.be>

    Permission is hereby granted, free of charge, to any person obtaining a copy
    of this software and associated documentation files (the "Software"), to deal
    in the Software without restriction, including without limitation the rights
    to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
    copies of the Software, and to permit persons to whom the Software is
    furnished to do so, subject to the following conditions:

    The above copyright notice and this permission notice shall be included in
    all copies or substantial portions of the Software.

    THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
    IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
    FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
    AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
    LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
    OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
    THE SOFTWARE.
