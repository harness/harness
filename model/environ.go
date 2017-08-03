package model

import (
	"errors"
)

var (
	errEnvironNameInvalid  = errors.New("Invalid Environment Variable Name")
	errEnvironValueInvalid = errors.New("Invalid Environment Variable Value")
)

// EnvironService defines a service for managing environment variables.
type EnvironService interface {
	EnvironList(*Repo) ([]*Environ, error)
}

// EnvironStore persists environment information to storage.
type EnvironStore interface {
	EnvironList(*Repo) ([]*Environ, error)
}

// Environ represents an environment variable.
// swagger:model environ
type Environ struct {
	ID    int64  `json:"id"              meddler:"env_id,pk"`
	Name  string `json:"name"            meddler:"env_name"`
	Value string `json:"value,omitempty" meddler:"env_value"`
}

// Validate validates the required fields and formats.
func (e *Environ) Validate() error {
	switch {
	case len(e.Name) == 0:
		return errEnvironNameInvalid
	case len(e.Value) == 0:
		return errEnvironValueInvalid
	default:
		return nil
	}
}

// Copy makes a copy of the environment variable without the value.
func (e *Environ) Copy() *Environ {
	return &Environ{
		ID:   e.ID,
		Name: e.Name,
	}
}
