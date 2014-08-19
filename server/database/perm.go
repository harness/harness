package database

import (
	"time"

	"github.com/drone/drone/shared/model"
	"github.com/jinzhu/gorm"
)

type PermManager interface {
	// Grant will grant the user read, write and admin persmissions
	// to the specified repository.
	Grant(u *model.User, r *model.Repo, read, write, admin bool) error

	// Revoke will revoke all user permissions to the specified repository.
	Revoke(u *model.User, r *model.Repo) error

	// Find returns the user's permission to access the specified repository.
	Find(u *model.User, r *model.Repo) *model.Perm

	// Read returns true if the specified user has read
	// access to the repository.
	Read(u *model.User, r *model.Repo) (bool, error)

	// Write returns true if the specified user has write
	// access to the repository.
	Write(u *model.User, r *model.Repo) (bool, error)

	// Admin returns true if the specified user is an
	// administrator of the repository.
	Admin(u *model.User, r *model.Repo) (bool, error)

	// Member returns true if the specified user is a
	// collaborator on the repository.
	Member(u *model.User, r *model.Repo) (bool, error)
}

// permManager manages user permissions to access repositories.
type permManager struct {
	ORM *gorm.DB
}

// NewManager initiales a new PermManager intended to
// manage user permission and access control.
func NewPermManager(db *gorm.DB) PermManager {
	return &permManager{ORM: db}
}

// Grant will grant the user read, write and admin persmissions
// to the specified repository.
func (db *permManager) Grant(u *model.User, r *model.Repo, read bool, write bool, admin bool) error {
	// attempt to get existing permissions from the database
	perm, err := db.find(u, r)
	if err != nil && err != gorm.RecordNotFound {
		return err
	}

	// if this is a new permission set the user ID,
	// repository ID and created timestamp.
	if perm.Id == 0 {
		perm.UserId = u.Id
		perm.RepoId = r.Id
		perm.Created = time.Now().Unix()
	}

	// set all the permission values
	perm.Read = read
	perm.Write = write
	perm.Admin = admin
	perm.Updated = time.Now().Unix()

	// update the database
	if perm.Id == 0 {
		return db.ORM.Create(perm).Error
	} else {
		// Fix bool update https://github.com/jinzhu/gorm/issues/202#issuecomment-52582525
		return db.ORM.Table("perms").Where(model.Perm{Id: perm.Id}).Update(perm).
			Update(map[string]interface{}{"Read": read, "Write": write, "Admin": admin}).Error
	}
}

// Revoke will revoke all user permissions to the specified repository.
func (db *permManager) Revoke(u *model.User, r *model.Repo) error {
	return db.ORM.Table("perms").Delete(model.Perm{UserId: u.Id, RepoId: r.Id}).Error
}

func (db *permManager) Find(u *model.User, r *model.Repo) *model.Perm {
	// if the user is a gues they should only be granted
	// read access to public repositories.
	switch {
	case u == nil && r.Private:
		return &model.Perm{
			Guest: true,
			Read:  false,
			Write: false,
			Admin: false}
	case u == nil && !r.Private:
		return &model.Perm{
			Guest: true,
			Read:  true,
			Write: false,
			Admin: false}
	}

	// if the user is authenticated we'll retireive the
	// permission details from the database.
	perm, err := db.find(u, r)
	if err != nil && perm.Id != 0 {
		return perm
	}

	switch {
	// if the user is a system admin grant super access.
	case u.Admin == true:
		perm.Read = true
		perm.Write = true
		perm.Admin = true
		perm.Guest = true

	// if the repo is public, grant read access only.
	case r.Private == false:
		perm.Read = true
		perm.Guest = true
	}

	return perm
}

func (db *permManager) Read(u *model.User, r *model.Repo) (bool, error) {
	return db.Find(u, r).Read, nil
}

func (db *permManager) Write(u *model.User, r *model.Repo) (bool, error) {
	return db.Find(u, r).Write, nil
}

func (db *permManager) Admin(u *model.User, r *model.Repo) (bool, error) {
	return db.Find(u, r).Admin, nil
}

func (db *permManager) Member(u *model.User, r *model.Repo) (bool, error) {
	perm := db.Find(u, r)
	return perm.Read && !perm.Guest, nil
}

func (db *permManager) find(u *model.User, r *model.Repo) (*model.Perm, error) {
	perm := model.Perm{}

	err := db.ORM.Table("perms").Where(model.Perm{UserId: u.Id, RepoId: r.Id}).First(&perm).Error
	return &perm, err
}
