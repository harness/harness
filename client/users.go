package client

import (
	"fmt"

	"github.com/drone/drone/shared/model"
)

type UserService struct {
	*Client
}

// GET /v1/users/{host}/{login}
func (s *UserService) Get(remote, login string) (*model.User, error) {
	var path = fmt.Sprintf("/v1/users/%s/%s", remote, login)
	var user = model.User{}
	var err = s.run("GET", path, nil, &user)
	return &user, err
}

// GET /v1/user
func (s *UserService) GetCurrent() (*model.User, error) {
	var user = model.User{}
	var err = s.run("GET", "/v1/user", nil, &user)
	return &user, err
}

// POST /v1/users/{host}/{login}
func (s *UserService) Create(remote, login string) (*model.User, error) {
	var path = fmt.Sprintf("/v1/users/%s/%s", remote, login)
	var user = model.User{}
	var err = s.run("POST", path, nil, &user)
	return &user, err
}

// DELETE /v1/users/{host}/{login}
func (s *UserService) Delete(remote, login string) error {
	var path = fmt.Sprintf("/v1/users/%s/%s", remote, login)
	return s.run("DELETE", path, nil, nil)
}

// GET /v1/users
func (s *UserService) List() ([]*model.User, error) {
	var users []*model.User
	var err = s.run("GET", "/v1/users", nil, &users)
	return users, err
}
