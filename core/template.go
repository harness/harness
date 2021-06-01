// Copyright 2019 Drone IO, Inc.
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

package core

import (
	"context"

	"github.com/drone/drone/handler/api/errors"
)

var (
	errTemplateNameInvalid = errors.New("No Template Name Provided")
	errTemplateDataInvalid = errors.New("No Template Data Provided")
)

type (
	TemplateArgs struct {
		Kind string
		Load string
		Data map[string]interface{}
	}

	Template struct {
		Id        int64  `json:"id,omitempty"`
		Name      string `json:"name,omitempty"`
		Namespace string `json:"namespace,omitempty"`
		Data      string `json:"data,omitempty"`
		Created   int64  `json:"created,omitempty"`
		Updated   int64  `json:"updated,omitempty"`
	}

	// TemplateStore manages repository templates.
	TemplateStore interface {
		// List returns template list at org level
		List(ctx context.Context, namespace string) ([]*Template, error)

		// ListAll returns templates list from the datastore.
		ListAll(ctx context.Context) ([]*Template, error)

		// Find returns a template from the datastore.
		Find(ctx context.Context, id int64) (*Template, error)

		// FindName returns a template from the data store
		FindName(ctx context.Context, name string, namespace string) (*Template, error)

		// Create persists a new template to the datastore.
		Create(ctx context.Context, template *Template) error

		// Update persists an updated template to the datastore.
		Update(ctx context.Context, template *Template) error

		// Delete deletes a template from the datastore.
		Delete(ctx context.Context, template *Template) error
	}
)

// Validate validates the required fields and formats.
func (s *Template) Validate() error {
	switch {
	case len(s.Name) == 0:
		return errTemplateNameInvalid
	case len(s.Data) == 0:
		return errTemplateDataInvalid
	default:
		return nil
	}
}
