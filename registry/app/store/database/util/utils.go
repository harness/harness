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
	"fmt"
	"strings"

	"github.com/harness/gitness/registry/utils"
)

func GetEmptySQLString(str string) sql.NullString {
	if utils.IsEmpty(str) {
		return sql.NullString{String: str, Valid: false}
	}
	return sql.NullString{String: str, Valid: true}
}

func GetEmptySQLInt64(i int64) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{Int64: i, Valid: false}
	}
	return sql.NullInt64{Int64: i, Valid: true}
}

func ConstructQuery(query string, args []interface{}) string {
	var builder strings.Builder
	argIndex := 0

	for i := 0; i < len(query); i++ {
		if query[i] == '?' && argIndex < len(args) {
			arg := args[argIndex]
			argIndex++

			// Convert the argument to a SQL-safe string
			var argStr string
			switch v := arg.(type) {
			case string:
				argStr = fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''")) // Escape single quotes in strings
			case int, int64, float64:
				argStr = fmt.Sprintf("%v", v)
			case bool:
				argStr = fmt.Sprintf("%t", v)
			default:
				argStr = fmt.Sprintf("'%v'", v)
			}

			builder.WriteString(argStr)
		} else {
			builder.WriteByte(query[i])
		}
	}

	return builder.String()
}

// FormatQuery is a helper function to interpolate parameters into the query.
func FormatQuery(query string, params []interface{}) string {
	for i, param := range params {
		placeholder := fmt.Sprintf("$%d", i+1)
		var value string
		switch v := param.(type) {
		case string:
			value = fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
		case []string:
			quotedValues := make([]string, len(v))
			for i, s := range v {
				quotedValues[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(s, "'", "''"))
			}
			value = fmt.Sprintf("ARRAY[%s]", strings.Join(quotedValues, ", "))
		default:
			value = fmt.Sprintf("%v", v)
		}
		query = strings.Replace(query, placeholder, value, 1)
	}
	return query
}

func SafeIntToUInt64(i int) uint64 {
	if i < 0 {
		return 0
	}
	return uint64(i)
}
