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

package remote

import (
	"golang.org/x/net/context"
)

const key = "remote"

// Setter defines a context that enables setting values.
type Setter interface {
	Set(string, interface{})
}

// FromContext returns the Remote associated with this context.
func FromContext(c context.Context) Remote {
	return c.Value(key).(Remote)
}

// ToContext adds the Remote to this context if it supports
// the Setter interface.
func ToContext(c Setter, r Remote) {
	c.Set(key, r)
}
