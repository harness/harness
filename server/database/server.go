package database

import (
	"github.com/drone/drone/shared/model"
	"github.com/jinzhu/gorm"
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
	ORM *gorm.DB
}

// NewServerManager initiales a new ServerManager intended to
// manage and persist servers.
func NewServerManager(db *gorm.DB) ServerManager {
	return &serverManager{ORM: db}
}

func (db *serverManager) Find(id int64) (*model.Server, error) {
	server := model.Server{}

	err := db.ORM.First(&server, id).Error
	return &server, err
}

func (db *serverManager) FindName(name string) (*model.Server, error) {
	server := model.Server{}

	err := db.ORM.Where(model.Server{Name: name}).First(&server).Error
	return &server, err
}

func (db *serverManager) FindSMTP() (*model.SMTPServer, error) {
	smtp := model.SMTPServer{}

	err := db.ORM.Table("smtp").Where(model.SMTPServer{Id: 1}).First(&smtp).Error
	return &smtp, err
}

func (db *serverManager) List() ([]*model.Server, error) {
	var servers []*model.Server

	err := db.ORM.Find(&servers).Error
	return servers, err
}

func (db *serverManager) Insert(server *model.Server) error {
	return db.ORM.Create(server).Error
}

func (db *serverManager) Update(server *model.Server) error {
	return db.ORM.Save(server).Error
}

func (db *serverManager) UpdateSMTP(server *model.SMTPServer) error {
	if server.Id == 0 {
		return db.ORM.Table("smtp").Create(server).Error
	} else {
		return db.ORM.Table("smtp").Where(model.SMTPServer{Id: server.Id}).Update(server).Error
	}
}

func (db *serverManager) Delete(server *model.Server) error {
	return db.ORM.Table("servers").Delete(server).Error
}
