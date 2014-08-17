package database

import (
	"time"

	"github.com/drone/drone/shared/model"
	"github.com/jinzhu/gorm"
)

type UserManager interface {
	// Find finds the User by ID.
	Find(id int64) (*model.User, error)

	// FindLogin finds the User by remote login.
	FindLogin(remote, login string) (*model.User, error)

	// FindToken finds the User by token.
	FindToken(token string) (*model.User, error)

	// List finds all registered users of the system.
	List() ([]*model.User, error)

	// Insert persists the User to the datastore.
	Insert(user *model.User) error

	// Update persists changes to the User to the datastore.
	Update(user *model.User) error

	// Delete removes the User from the datastore.
	Delete(user *model.User) error

	// Exist returns true if Users exist in the system.
	Exist() bool
}

// userManager manages a list of users in a SQL database.
type userManager struct {
	ORM *gorm.DB
}

// NewUserManager initiales a new UserManager intended to
// manage and persist commits.
func NewUserManager(db *gorm.DB) UserManager {
	return &userManager{ORM: db}
}

func (db *userManager) Find(id int64) (*model.User, error) {
	person := model.User{}

	err := db.ORM.Table("users").Where(&model.User{Id: id}).First(&person).Error
	return &person, err
}

func (db *userManager) FindLogin(remote, login string) (*model.User, error) {
	person := model.User{}

	err := db.ORM.Table("users").Where(&model.User{Remote: remote, Login: login}).First(&person).Error
	return &person, err
}

func (db *userManager) FindToken(token string) (*model.User, error) {
	person := model.User{}

	err := db.ORM.Table("users").Where(&model.User{Token: token}).First(&person).Error
	return &person, err
}

func (db *userManager) List() ([]*model.User, error) {
	var users []*model.User

	err := db.ORM.Table("users").Find(&users).Error
	return users, err
}

func (db *userManager) Insert(user *model.User) error {
	user.Created = time.Now().Unix()
	user.Updated = time.Now().Unix()

	return db.ORM.Table("users").Create(user).Error
}

func (db *userManager) Update(user *model.User) error {
	user.Updated = time.Now().Unix()
	return db.ORM.Table("users").Where(model.User{Id: user.Id}).Update(user).Error
}

func (db *userManager) Delete(user *model.User) error {
	return db.ORM.Table("users").Delete(user).Error
}

func (db *userManager) Exist() bool {
	var result int
	db.ORM.Table("users").Select("0").Limit("1").Scan(&result)
	return result == 1
}
