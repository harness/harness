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

package database

import (
	"strings"
)

const (
	PostgresDriverName = "postgres"
	SqliteDriverName   = "sqlite3"
)

// PartialMatch builds a string pair that can be passed as a parameter to squirrel's Where() function
// for a SQL "LIKE" expression. Besides surrounding the input value with '%' wildcard characters for a partial match,
// this function also escapes the '_' and '%' metacharacters supported in SQL "LIKE" expressions.
// The "ESCAPE" clause isn't needed for Postgres, but is necessary for SQLite.
// It will be used only if '_' and '%' are present in the value string.
//
// See:
// https://www.postgresql.org/docs/current/functions-matching.html#FUNCTIONS-LIKE
// https://www.sqlite.org/lang_expr.html#the_like_glob_regexp_match_and_extract_operators
func PartialMatch(column, value string) (string, string) {
	var (
		n       int
		escaped bool
	)

	if n, value = len(value), strings.ReplaceAll(value, `\`, `\\`); n < len(value) {
		escaped = true
	}
	if n, value = len(value), strings.ReplaceAll(value, "_", `\_`); n < len(value) {
		escaped = true
	}
	if n, value = len(value), strings.ReplaceAll(value, "%", `\%`); n < len(value) {
		escaped = true
	}

	sb := strings.Builder{}
	sb.WriteString("LOWER(")
	sb.WriteString(column)
	sb.WriteString(") LIKE '%' || LOWER(?) || '%'")
	if escaped {
		sb.WriteString(` ESCAPE '\'`)
	}

	return sb.String(), value
}
