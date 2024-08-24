//  Copyright 2023 Harness, Inc.
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

package util

import (
	"database/sql"

	"github.com/harness/gitness/registry/app/pkg/commons"
)

func GetEmptySQLString(str string) sql.NullString {
	if commons.IsEmpty(str) {
		return sql.NullString{String: str, Valid: false}
	}
	return sql.NullString{String: str, Valid: true}
}

func GetEmptySQLInt32(i int) sql.NullInt32 {
	if i == 0 {
		return sql.NullInt32{Int32: int32(i), Valid: false}
	}
	return sql.NullInt32{Int32: int32(i), Valid: true}
}
