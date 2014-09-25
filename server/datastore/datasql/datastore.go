package datasql

import (
	"database/sql"

	"github.com/drone/drone/server/blobstore/blobsql"
	"github.com/drone/drone/server/datastore"
	"github.com/drone/drone/shared/model"

	"github.com/astaxie/beego/orm"
	_ "github.com/mattn/go-sqlite3"
)

const (
	driverPostgres = "postgres"
	driverSqlite   = "sqlite3"
	driverMysql    = "mysql"
	databaseName   = "default"
)

// Connect is a helper function that establishes a new
// database connection and auto-generates the database
// schema. If the database already exists, it will perform
// and update as needed.
func Connect(driver, datasource string) (*sql.DB, error) {
	defer orm.ResetModelCache()
	orm.RegisterDriver(driverSqlite, orm.DR_Sqlite)
	orm.RegisterDataBase(databaseName, driver, datasource)
	orm.RegisterModel(new(model.User))
	orm.RegisterModel(new(model.Perm))
	orm.RegisterModel(new(model.Repo))
	orm.RegisterModel(new(model.Commit))
	orm.RegisterModel(new(blobsql.Blob))
	var err = orm.RunSyncdb(databaseName, true, true)
	if err != nil {
		return nil, err
	}
	return orm.GetDB(databaseName)
}

// New returns a new DataStore
func New(db *sql.DB) datastore.Datastore {
	return struct {
		*Userstore
		*Permstore
		*Repostore
		*Commitstore
	}{
		NewUserstore(db),
		NewPermstore(db),
		NewRepostore(db),
		NewCommitstore(db),
	}
}
