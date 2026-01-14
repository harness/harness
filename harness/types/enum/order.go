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
	"strings"
)

// Order defines the sort order.
type Order int

// Order enumeration.
const (
	OrderDefault Order = iota
	OrderAsc
	OrderDesc
)

// String returns the Order as a string.
func (e Order) String() string {
	switch e {
	case OrderDesc:
		return desc
	case OrderAsc:
		return asc
	case OrderDefault:
		return desc
	default:
		return undefined
	}
}

// ParseOrder parses the order string and returns
// an order enumeration.
func ParseOrder(s string) Order {
	switch strings.ToLower(s) {
	case asc, ascending:
		return OrderAsc
	case desc, descending:
		return OrderDesc
	default:
		return OrderDefault
	}
}
