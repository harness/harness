# Building SQL DDL into Drone

These folders contain the code for the different of databases that drone can use. They contain the SQL necessary to create the necessary tables and migrate between versions (IE the DDL). This SQL is generated into a go file and included as part of the Drone binary.

## Making a changes to the database DDL

Any new changes to the database structure are always put into a new SQL file. Follow the naming scheme in the `store/shared/migrate/<db>/files` of the SQL files by incrementing the number file name and give it a good description of what changes are being made.

Changes will need to be implemented for all supported databases, making similar changes for eg Mysql/Postgres/Sqllite.

**NB** Any changes to the database structure will need to be reflected for the relevant `struct` in the `core` directory. Changing the objects in the `store` directory for the ORM. Finally Possibly in the repositories github.com/drone/drone-go and  github.com/drone/runner-go.

## Generating Go from the SQL files

To generate the go files you will need to install the golang command line tool `Togo` so it is on your users PATH.

### Steps to install Togo

``` bash
# in your workspace
git clone git@github.com:bradrydzewski/togo.git
cd togo
go get github.com/bradrydzewski/togo
```

### Generating go DDL

Enter the desired database's implementation folder, and run the following. It will update the `ddl_gen.go` file.

``` bash
go generate
```
