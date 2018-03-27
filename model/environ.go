// Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
