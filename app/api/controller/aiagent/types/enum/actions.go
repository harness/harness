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

package enum

import (
	"github.com/harness/gitness/types/enum"
)

type Action string

const (
	Accept     Action = "ACCEPT"
	Regenerate Action = "REGENERATE"
)

func (a Action) Sanitize() (Action, bool) {
	return enum.Sanitize(a, GetAllActions)
}

func GetAllActions() ([]Action, Action) {
	return Actions, ""
}

var Actions = ([]Action{
	Accept,
	Regenerate,
})
