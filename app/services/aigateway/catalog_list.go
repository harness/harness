// Copyright 2026 Harness, Inc.
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

package aigateway

import "sort"

// Models returns every free model currently in the catalog (including quarantined ones), sorted by ID.
func (c *Catalog) Models() []Model {
	c.mu.RLock()
	defer c.mu.RUnlock()

	res := make([]Model, 0, len(c.models))
	for _, m := range c.models {
		res = append(res, m)
	}
	sort.Slice(res, func(i, j int) bool { return res[i].ID < res[j].ID })
	return res
}
