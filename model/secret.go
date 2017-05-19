package model

import (
	"errors"
	"path/filepath"
)

var (
	errSecretNameInvalid  = errors.New("Invalid Secret Name")
	errSecretValueInvalid = errors.New("Invalid Secret Value")
)

// SecretService defines a service for managing secrets.
type SecretService interface {
	SecretFind(*Repo, string) (*Secret, error)
	SecretList(*Repo) ([]*Secret, error)
	SecretListBuild(*Repo, *Build) ([]*Secret, error)
	SecretCreate(*Repo, *Secret) error
	SecretUpdate(*Repo, *Secret) error
	SecretDelete(*Repo, string) error
}

// SecretStore persists secret information to storage.
type SecretStore interface {
	SecretFind(*Repo, string) (*Secret, error)
	SecretList(*Repo) ([]*Secret, error)
	SecretCreate(*Secret) error
	SecretUpdate(*Secret) error
	SecretDelete(*Secret) error
}

// Secret represents a secret variable, such as a password or token.
// swagger:model registry
type Secret struct {
	ID         int64    `json:"id"              meddler:"secret_id,pk"`
	RepoID     int64    `json:"-"               meddler:"secret_repo_id"`
	Name       string   `json:"name"            meddler:"secret_name"`
	Value      string   `json:"value,omitempty" meddler:"secret_value"`
	Images     []string `json:"image"           meddler:"secret_images,json"`
	Events     []string `json:"event"           meddler:"secret_events,json"`
	SkipVerify bool     `json:"-"               meddler:"secret_skip_verify"`
	Conceal    bool     `json:"-"               meddler:"secret_conceal"`
}

// Match returns true if an image and event match the restricted list.
func (s *Secret) Match(event string) bool {
	if len(s.Events) == 0 {
		return true
	}
	for _, pattern := range s.Events {
		if match, _ := filepath.Match(pattern, event); match {
			return true
		}
	}
	return false
}

// Validate validates the required fields and formats.
func (s *Secret) Validate() error {
	switch {
	case len(s.Name) == 0:
		return errSecretNameInvalid
	case len(s.Value) == 0:
		return errSecretValueInvalid
	default:
		return nil
	}
}

// Copy makes a copy of the secret without the value.
func (s *Secret) Copy() *Secret {
	return &Secret{
		ID:     s.ID,
		RepoID: s.RepoID,
		Name:   s.Name,
		Images: s.Images,
		Events: s.Events,
	}
}
