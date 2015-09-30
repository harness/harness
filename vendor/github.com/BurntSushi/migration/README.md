Package migration for Golang automatically handles versioning of a database 
schema by applying a series of migrations supplied by the client. It uses 
features only from the database/sql package, so it tries to be driver 
independent. However, to track the version of the database, it is necessary to 
execute some SQL. I've made an effort to keep those queries simple, but if they 
don't work with your database, you may override them.

This package works by applying a series of migrations to a database. Once a 
migration is created, it should never be changed. Every time a database is 
opened with this package, all necessary migrations are executed in a single 
transaction. If any part of the process fails, an error is returned and the 
transaction is rolled back so that the database is left untouched. (Note that 
for this to be useful, you'll need to use a database that supports rolling back 
changes to your schema. Notably, MySQL does not support this, although SQLite 
and PostgreSQL do.)

The version of a database is defined as the number of migrations applied to it.


### Installation

If you have Go installed and
[your GOPATH is setup](http://golang.org/doc/code.html#GOPATH), then 
`migration` can be installed with `go get`:

    go get github.com/BurntSushi/migration


### Documentation

Documentation is available at
[godoc.org/github.com/BurntSushi/migration](http://godoc.org/github.com/BurntSushi/migration).


### Unstable

At the moment, I'm still experimenting with the public API, so I may still 
introduce breaking changes. In general though, I am happy with the overall 
architecture.

