package database

import (
	"database/sql"

	"github.com/drone/drone/shared/model"
	"github.com/russross/meddler"
)

type ServerManager interface {
	// Find finds the Server by ID.
	Find(id int64) (*model.Server, error)

	// FindName finds the Server by name.
	FindName(name string) (*model.Server, error)

	// FindName finds the Server by name.
	FindSMTP() (*model.SMTPServer, error)

	// List finds all registered Servers of the system.
	List() ([]*model.Server, error)

	// Insert persists the Server to the datastore.
	Insert(server *model.Server) error

	// Update persists changes to the Server to the datastore.
	Update(server *model.Server) error

	// UpdateSMTP persists changes to the SMTP Server to the datastore.
	UpdateSMTP(server *model.SMTPServer) error

	// Delete removes the Server from the datastore.
	Delete(server *model.Server) error
}

// serverManager manages a list of users in a SQL database.
type serverManager struct {
	*sql.DB
}

// SQL query to retrieve a Server by remote login.
const findServerQuery = `
SELECT *
FROM servers
WHERE server_name=?
LIMIT 1
`

// SQL query to retrieve a list of all Servers.
const listServerQuery = `
SELECT *
FROM servers
`

// SQL statement to delete a Server by ID.
const deleteServerStmt = `
DELETE FROM servers WHERE server_id=?
`

// NewServerManager initiales a new ServerManager intended to
// manage and persist servers.
func NewServerManager(db *sql.DB) ServerManager {
	return &serverManager{db}
}

func (db *serverManager) Find(id int64) (*model.Server, error) {
	dst := model.Server{}
	err := meddler.Load(db, "servers", &dst, id)
	return &dst, err
}

func (db *serverManager) FindName(name string) (*model.Server, error) {
	dst := model.Server{}
	err := meddler.QueryRow(db, &dst, findServerQuery, name)
	return &dst, err
}

func (db *serverManager) FindSMTP() (*model.SMTPServer, error) {
	dst := model.SMTPServer{}
	err := meddler.Load(db, "smtp", &dst, 1)
	if err != nil && err != sql.ErrNoRows {
		return &dst, err
	}
	return &dst, nil
}

func (db *serverManager) List() ([]*model.Server, error) {
	var dst []*model.Server
	err := meddler.QueryAll(db, &dst, listServerQuery)
	return dst, err
}

func (db *serverManager) Insert(server *model.Server) error {
	return meddler.Insert(db, "servers", server)
}

func (db *serverManager) Update(server *model.Server) error {
	return meddler.Update(db, "servers", server)
}

func (db *serverManager) UpdateSMTP(server *model.SMTPServer) error {
	server.ID = 0
	meddler.Insert(db, "smtp", server)
	server.ID = 1
	return meddler.Update(db, "smtp", server)
}

func (db *serverManager) Delete(server *model.Server) error {
	_, err := db.Exec(deleteServerStmt, server.ID)
	return err
}
