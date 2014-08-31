package database

import (
	"github.com/drone/drone/shared/model"
	"github.com/jinzhu/gorm"
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
	ORM *gorm.DB
}

// NewRemoteManager initiales a new RemoteManager intended to
// manage and persist servers.
func NewRemoteManager(db *gorm.DB) RemoteManager {
	return &remoteManager{ORM: db}
}

func (db *remoteManager) Find(id int64) (*model.Remote, error) {
	remote := model.Remote{}

	err := db.ORM.First(&remote, id).Error
	return &remote, err
}

func (db *remoteManager) FindHost(host string) (*model.Remote, error) {
	remote := model.Remote{}

	err := db.ORM.Where(model.Remote{Host: host}).First(&remote).Error
	return &remote, err
}

func (db *remoteManager) FindType(t string) (*model.Remote, error) {
	remote := model.Remote{}

	err := db.ORM.Where(model.Remote{Type: t}).First(&remote).Error
	return &remote, err
}

func (db *remoteManager) List() ([]*model.Remote, error) {
	var remotes []*model.Remote

	err := db.ORM.Find(&remotes).Error
	return remotes, err
}

func (db *remoteManager) Insert(remote *model.Remote) error {
	return db.ORM.Create(remote).Error
}

func (db *remoteManager) Update(remote *model.Remote) error {
	return db.ORM.Save(remote).Error
}

func (db *remoteManager) Delete(remote *model.Remote) error {
	return db.ORM.Delete(remote).Error
}
