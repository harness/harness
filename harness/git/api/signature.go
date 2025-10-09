// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"fmt"
	"time"

	"github.com/harness/gitness/errors"
)

// Signature represents the Author or Committer information.
type Signature struct {
	Identity Identity
	// When is the timestamp of the Signature.
	When time.Time
}

func (s Signature) String() string {
	return fmt.Sprintf("%s <%s>", s.Identity.Name, s.Identity.Email)
}

type Identity struct {
	Name  string
	Email string
}

func (i Identity) String() string {
	return fmt.Sprintf("%s <%s>", i.Name, i.Email)
}

func (i Identity) Validate() error {
	if i.Name == "" {
		return errors.InvalidArgument("identity name is mandatory")
	}

	if i.Email == "" {
		return errors.InvalidArgument("identity email is mandatory")
	}

	return nil
}
