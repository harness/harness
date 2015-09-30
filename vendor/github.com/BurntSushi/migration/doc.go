/*
Package migration automatically handles versioning of a database 
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
*/
package migration
