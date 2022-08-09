// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

import "strings"

// Order defines the sorder order.
type Order int

// Order enumeration.
const (
	OrderDefault Order = iota
	OrderAsc
	OrderDesc
)

// String returns the Order as a string.
func (e Order) String() (s string) {
	switch e {
	case OrderAsc:
		return "asc"
	case OrderDesc:
		return "desc"
	default:
		return "asc" // ascending by default?
	}
}

// ParseOrder parses the order string and returns
// an order enumeration.
func ParseOrder(s string) Order {
	switch strings.ToLower(s) {
	case "asc", "ascending":
		return OrderAsc
	case "desc", "descending":
		return OrderDesc
	default:
		return OrderDefault
	}
}
