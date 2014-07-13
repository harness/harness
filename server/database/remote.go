package database

import (
	"database/sql"

	"github.com/drone/drone/shared/model"
	"github.com/russross/meddler"
)

type RemoteManager interface {
	// Find finds the Remote by ID.
	Find(id int64) (*model.Remote, error)

	// FindHost finds the Remote by hostname.
	FindHost(name string) (*model.Remote, error)

	// FindHost finds the Remote by type.
	FindType(t string) (*model.Remote, error)

	// List finds all registered Remotes of the system.
	List() ([]*model.Remote, error)

	// Insert persists the Remotes to the datastore.
	Insert(server *model.Remote) error

	// Update persists changes to the Remotes to the datastore.
	Update(server *model.Remote) error

	// Delete removes the Remotes from the datastore.
	Delete(server *model.Remote) error
}

// remoteManager manages a list of remotes in a SQL database.
type remoteManager struct {
	*sql.DB
}

// SQL query to retrieve a Remote by remote login.
const findRemoteQuery = `
SELECT *
FROM remotes
WHERE remote_host=?
LIMIT 1
`

// SQL query to retrieve a Remote by remote login.
const findRemoteTypeQuery = `
SELECT *
FROM remotes
WHERE remote_type=?
LIMIT 1
`

// SQL query to retrieve a list of all Remotes.
const listRemoteQuery = `
SELECT *
FROM remotes
ORDER BY remote_type
`

// SQL statement to delete a Remote by ID.
const deleteRemoteStmt = `
DELETE FROM remotes WHERE remote_id=?
`

// NewRemoteManager initiales a new RemoteManager intended to
// manage and persist servers.
func NewRemoteManager(db *sql.DB) RemoteManager {
	return &remoteManager{db}
}

func (db *remoteManager) Find(id int64) (*model.Remote, error) {
	dst := model.Remote{}
	err := meddler.Load(db, "remotes", &dst, id)
	return &dst, err
}

func (db *remoteManager) FindHost(host string) (*model.Remote, error) {
	dst := model.Remote{}
	err := meddler.QueryRow(db, &dst, findRemoteQuery, host)
	return &dst, err
}

func (db *remoteManager) FindType(t string) (*model.Remote, error) {
	dst := model.Remote{}
	err := meddler.QueryRow(db, &dst, findRemoteTypeQuery, t)
	return &dst, err
}

func (db *remoteManager) List() ([]*model.Remote, error) {
	var dst []*model.Remote
	err := meddler.QueryAll(db, &dst, listRemoteQuery)
	return dst, err
}

func (db *remoteManager) Insert(remote *model.Remote) error {
	return meddler.Insert(db, "remotes", remote)
}

func (db *remoteManager) Update(remote *model.Remote) error {
	return meddler.Update(db, "remotes", remote)
}

func (db *remoteManager) Delete(remote *model.Remote) error {
	_, err := db.Exec(deleteRemoteStmt, remote.ID)
	return err
}
