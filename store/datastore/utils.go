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

package datastore

import (
	"strconv"

	"github.com/russross/meddler"
)

// rebind is a helper function that changes the sql
// bind type from ? to $ for postgres queries.
func rebind(query string) string {
	if meddler.Default != meddler.PostgreSQL {
		return query
	}

	qb := []byte(query)
	// Add space enough for 5 params before we have to allocate
	rqb := make([]byte, 0, len(qb)+5)
	j := 1
	for _, b := range qb {
		switch b {
		case '?':
			rqb = append(rqb, '$')
			for _, b := range strconv.Itoa(j) {
				rqb = append(rqb, byte(b))
			}
			j++
		case '`':
			rqb = append(rqb, ' ')
		default:
			rqb = append(rqb, b)
		}
	}
	return string(rqb)
}
