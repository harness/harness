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

package types

// ListQueryFilter has pagination related info and a query param.
type ListQueryFilter struct {
	Pagination
	Query string `json:"query"`
}

type CreatedFilter struct {
	CreatedGt int64 `json:"created_gt"`
	CreatedLt int64 `json:"created_lt"`
}

type UpdatedFilter struct {
	UpdatedGt int64 `json:"updated_gt"`
	UpdatedLt int64 `json:"updated_lt"`
}

type EditedFilter struct {
	EditedGt int64 `json:"edited_gt"`
	EditedLt int64 `json:"edited_lt"`
}
