Meddler
=======

Meddler is a small toolkit to take some of the tedium out of moving data
back and forth between sql queries and structs.

It is not a complete ORM. It is intended to be lightweight way to add some
of the convenience of an ORM while leaving more control in the hands of the
programmer.

Package docs are available at:

* http://godoc.org/github.com/russross/meddler

The package is housed on github, and the README there has more info:

* http://github.com/russross/meddler

This is currently configured for SQLite, MySQL, and PostgreSQL, but it
can be configured for use with other databases. If you use it
successfully with a different database, please contact me and I will
add it to the list of pre-configured databases.

### DANGER

Meddler is still a work in progress, and additional
backward-incompatible changes to the API are likely. The most recent
change added support for multiple database types and made it easier
to switch between them. This is most likely to affect the way you
initialize the library to work with your database (see the install
section below).

Another recent update is the change to int64 for primary keys. This
matches the convention used in database/sql, and is more portable,
but it may require some minor changes to existing code.


Install
-------

The usual `go get` command will put it in your `$GOPATH`:

    go get github.com/russross/meddler

If you are only using one type of database, you should set Default
to match your database type, e.g.:

    meddler.Default = meddler.PostgreSQL

The default database is MySQL, so you should change it for anything
else. To use multiple databases within a single project, or to use a
database other than MySQL, PostgreSQL, or SQLite, see below.

Note: If you are using MySQL with the `github.com/go-sql-driver/mysql`
driver, you must set "parseTime=true" in the sql.Open call or the
time conversion meddlers will not work.


Why?
----

These are the features that set meddler apart from similar
libraries:

*   It uses standard database/sql types, and does not require
    special fields in your structs. This lets you use meddler
    selectively, without having to alter other database code already
    in your project. After creating meddler, I incorporated it into
    an existing project, and I was able to convert the code one
    struct and one query at a time.
*   It leaves query writing to you. It has convenience functions for
    simple INSERT/UPDATE/SELECT queries by integer primary key, but
    beyond that it stays out of query writing.
*   It supports on-the-fly data transformations. If you have a map
    or a slice in your struct, you can instruct meddler to
    encode/decode using JSON or Gob automatically. If you have time
    fields, you can have meddler automatically write them into the
    database as UTC, and convert them to the local time zone on
    reads.  These processors are called “meddlers”, because they
    meddle with the data instead of passing it through directly.
*   NULL fields in the database can be read as zero values in the
    struct, and zero values in the struct can be written as NULL
    values. This is not always the right thing to do, but it is
    often good enough and is much simpler than most alternatives.
*   It exposes low-level hooks for more complex situations. If you
    are writing a query that does not map well to the main helper
    functions, you can still get some help by using the lower-level
    functions to build your own helpers.


High-level functions
--------------------

Meddler does not create or alter tables. It just provides a little
glue to make it easier to read and write structs as SQL rows. Start
by annotating a struct:

``` go
type Person struct {
    ID      int       `meddler:"id,pk"`
    Name    string    `meddler:"name"`
    Age     int
    salary  int
    Created time.Time `meddler:"created,localtime"`
    Closed  time.Time `meddler:",localtimez"`
}
```

Notes about this example:

*   If the optional tag is provided, the first field is the database
    column name. Note that "Closed" does not provide a column name,
    so it will default to "Closed". Likewise, if there is no tag,
    the field name will be used.
*   ID is marked as the primary key. Currently only integer primary
    keys are supported. This is only relevant to Load, Save, Insert,
    and Update, a few of the higher-level functions that need to
    understand primary keys. Meddler assumes that pk fields have an
    autoincrement mechanism set in the database.
*   Age has a column name of "Age". A tag is only necessary when the
    column name is not the same as the field name, or when you need
    to select other options.
*   salary is not an exported field, so meddler does not see it. It
    will be ignored.
*   Created is marked with "localtime". This means that it will be
    converted to UTC when being saved, and back to the local time
    zone when being loaded.
*   Closed has a column name of "Closed", since the tag did not
    specify anything different. Closed is marked as "localtimez".
    This has the same properties as "localtime", except that the
    zero time will be saved in the database as a null column (and
    null values will be loaded as the zero time value).

Meddler provides a few high-level functions (note: DB is an
interface that works with a *sql.DB or a *sql.Tx):

*   Load(db DB, table string, dst interface{}, pk int64) error

    This loads a single record by its primary key. For example:

        elt := new(Person)
        err := meddler.Load(db, "person", elt, 15)

    db can be a *sql.DB or a *sql.Tx. The table is the name of the
    table, pk is the primary key value, and dst is a pointer to the
    struct where it should be stored.

    Note: this call requires that the struct have an integer primary
    key field marked.

*   Insert(db DB, table string, src interface{}) error

    This inserts a new row into the database. If the struct value
    has a primary key field, it must be zero (and will be omitted
    from the insert statement, prompting a default autoincrement
    value).

        elt := &Person{
            Name: "Alice",
            Age: 22,
            // ...
        }
        err := meddler.Insert(db, "person", elt)
        // elt.ID is updated to the value assigned by the database

*   Update(db DB, table string, src interface{}) error

    This updates an existing row. It must have a primary key, which
    must be non-zero.

    Note: this call requires that the struct have an integer primary
    key field marked.

*   Save(db DB, table string, src interface{}) error

    Pick Insert or Update automatically. If there is a non-zero
    primary key present, it uses Update, otherwise it uses Insert.

    Note: this call requires that the struct have an integer primary
    key field marked.

*   QueryRow(db DB, dst interface{}, query string, args ...interface) error

    Perform the given query, and scan the single-row result into
    dst, which must be a pointer to a struct.

    For example:

        elt := new(Person)
        err := meddler.QueryRow(db, elt, "select * from person where name = ?", "bob")

*   QueryAll(db DB, dst interface{}, query string, args ...interface) error

    Perform the given query, and scan the results into dst, which
    must be a pointer to a slice of struct pointers.

    For example:

        var people []*Person
        err := meddler.QueryAll(db, &people, "select * from person")

*   Scan(rows *sql.Rows, dst interface{}) error

    Scans a single row of data into a struct, complete with
    meddling. Can be called repeatedly to walk through all of the
    rows in a result set. Returns sql.ErrNoRows when there is no
    more data.

*   ScanRow(rows *sql.Rows, dst interface{}) error

    Similar to Scan, but guarantees that the rows object
    is closed when it returns. Also returns sql.ErrNoRows if there
    was no row.

*   ScanAll(rows *sql.Rows, dst interface{}) error

    Expects a pointer to a slice of structs/pointers to structs, and
    appends as many elements as it finds in the row set. Closes the
    row set when it is finished. Does not return sql.ErrNoRows on an
    empty set; instead it just does not add anything to the slice.

Note: all of these functions can also be used as methods on Database
objects. When used as package functions, they use the Default
Database object, which is MySQL unless you change it.


Meddlers
--------

A meddler is a handler that gets to meddle with a field before it is
saved, or when it is loaded. "localtime" and "localtimez" are
examples of built-in meddlers. The full list of built-in meddlers
includes:

*   identity: the default meddler, which does not do anything

*   localtime: for time.Time and *time.Time fields. Converts the
    value to UTC on save, and back to the local time zone on loads.
    To set your local time zone, make sure the TZ environment
    variable is set when your program is launched, or use something
    like:

        os.Setenv("TZ", "America/Denver")

    in your initial setup, before you start using time functions.

*   localtimez: same, but only for time.Time, and treats the zero
    time as a null field (converts both ways)

*   utctime: similar to localtime, but keeps the value in UTC on
    loads. This ensures that the time is always coverted to UTC on
    save, which is the sane way to save time values in a database.

*   utctimez: same, but with zero time means null.

*   zeroisnull: for other types where a zero value should be
    inserted as null, and null values should be read as zero values.
    Works for integer, unsigned integer, float, complex number, and
    string types. Note: not for pointer types.

*   json: marshals the field value into JSON when saving, and
    unmarshals on load.

*   jsongzip: same, but compresses using gzip on save, and
    uncompresses on load
    
*   gob: encodes the field value using Gob when saving, and
    decodes on load.

*   gobgzip: same, but compresses using gzip on save, and
    uncompresses on load
    
You can implement custom meddlers as well by implementing the
Meddler interface. See the existing implementations in medder.go for
examples.


Working with different database types
-------------------------------------

Meddler can work with multiple database types simultaneously.
Database-specific parameters are stored in a Database struct, and
structs are pre-defined for MySQL, PostgreSQL, and SQLite.

Instead of relying on the package-level functions, use the method
form on the appropriate database type, e.g.:

    err = meddler.PostgreSQL.Load(...)

instead of

    err = meddler.Load(...)

Or to save typing, define your own abbreviated name for each
database:

    ms := meddler.MySQL
    pg := meddler.PostgreSQL
    err = ms.Load(...)
    err = pg.QueryAll(...)

If you need a different database, create your own Database instance
with the appropriate parameters set. If everything works okay,
please contact me with the parameters you used so I can add the new
database to the pre-defined list.


Lower-level functions
---------------------

If you are using more complex queries and just want to reduce the
tedium of reading and writing values, there are some lower-level
helper functions as well. See the package docs for details, and
see the implementations of the higher-level functions to see how
they are used.


License
-------

Meddler is distributed under the BSD 2-Clause License. If this
license prevents you from using Meddler in your project, please
contact me and I will consider adding an additional license that is
better suited to your needs.

> Copyright © 2013 Russ Ross.
> All rights reserved.
> 
> Redistribution and use in source and binary forms, with or without
> modification, are permitted provided that the following conditions
> are met:
> 
> 1.  Redistributions of source code must retain the above copyright
>     notice, this list of conditions and the following disclaimer.
> 
> 2.  Redistributions in binary form must reproduce the above
>     copyright notice, this list of conditions and the following
>     disclaimer in the documentation and/or other materials provided with
>     the distribution.
> 
> THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
> "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
> LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS
> FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE
> COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT,
> INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING,
> BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
> LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
> CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
> LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN
> ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
> POSSIBILITY OF SUCH DAMAGE.
