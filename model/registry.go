package model

import "errors"

var (
	errRegistryAddressInvalid  = errors.New("Invalid Registry Address")
	errRegistryUsernameInvalid = errors.New("Invalid Registry Username")
	errRegistryPasswordInvalid = errors.New("Invalid Registry Password")
)

// RegistryService defines a service for managing registries.
type RegistryService interface {
	RegistryFind(*Repo, string) (*Registry, error)
	RegistryList(*Repo) ([]*Registry, error)
	RegistryCreate(*Repo, *Registry) error
	RegistryUpdate(*Repo, *Registry) error
	RegistryDelete(*Repo, string) error
}

// RegistryStore persists registry information to storage.
type RegistryStore interface {
	RegistryFind(*Repo, string) (*Registry, error)
	RegistryList(*Repo) ([]*Registry, error)
	RegistryCreate(*Registry) error
	RegistryUpdate(*Registry) error
	RegistryDelete(*Registry) error
}

// Registry represents a docker registry with credentials.
// swagger:model registry
type Registry struct {
	ID       int64  `json:"id"       meddler:"registry_id,pk"`
	RepoID   int64  `json:"-"        meddler:"registry_repo_id"`
	Address  string `json:"address"  meddler:"registry_addr"`
	Username string `json:"username" meddler:"registry_username"`
	Password string `json:"password" meddler:"registry_password"`
	Email    string `json:"email"    meddler:"registry_email"`
	Token    string `json:"token"    meddler:"registry_token"`
}

// Validate validates the registry information.
func (r *Registry) Validate() error {
	switch {
	case len(r.Address) == 0:
		return errRegistryAddressInvalid
	case len(r.Username) == 0:
		return errRegistryUsernameInvalid
	case len(r.Password) == 0:
		return errRegistryPasswordInvalid
	default:
		return nil
	}
}

// Copy makes a copy of the registry without the password.
func (r *Registry) Copy() *Registry {
	return &Registry{
		ID:       r.ID,
		RepoID:   r.RepoID,
		Address:  r.Address,
		Username: r.Username,
		Email:    r.Email,
		Token:    r.Token,
	}
}
