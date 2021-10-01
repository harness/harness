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
	"io"

	"github.com/drone/drone/handler/api/errors"
)

var (
	errCardStepInvalid   = errors.New("No Step ID Provided")
	errCardBuildInvalid  = errors.New("No Build ID Provided")
	errCardSchemaInvalid = errors.New("No Card Schema Has Been Provided")
)

type Card struct {
	Id     int64  `json:"id,omitempty"`
	Build  int64  `json:"build,omitempty"`
	Stage  int64  `json:"stage,omitempty"`
	Step   int64  `json:"step,omitempty"`
	Schema string `json:"schema,omitempty"`
}

type CardData struct {
	Id   int64  `json:"id,omitempty"`
	Data []byte `json:"card_data"`
}

// CardStore manages repository cards.
type CardStore interface {
	FindByBuild(ctx context.Context, build int64) ([]*Card, error)
	Find(ctx context.Context, step int64) (*Card, error)
	FindData(ctx context.Context, id int64) (io.ReadCloser, error)
	Create(ctx context.Context, card *Card, data io.ReadCloser) error
	Delete(ctx context.Context, id int64) error
}

// Validate validates the required fields and formats.
func (c *Card) Validate() error {
	switch {
	case c.Step == 0:
		return errCardStepInvalid
	case c.Build == 0:
		return errCardBuildInvalid
	case len(c.Schema) == 0:
		return errCardSchemaInvalid
	default:
		return nil
	}
}
