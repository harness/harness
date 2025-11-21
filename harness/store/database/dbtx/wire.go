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

package dbtx

import (
	"github.com/google/wire"
	"github.com/jmoiron/sqlx"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideAccessorTx,
	ProvideAccessor,
	ProvideTransactor,
)

// ProvideAccessorTx provides the most versatile database access interface.
// All DB queries and transactions can be performed.
func ProvideAccessorTx(db *sqlx.DB) AccessorTx {
	return New(db)
}

// ProvideAccessor provides the database access interface. All DB queries can be performed.
func ProvideAccessor(a AccessorTx) Accessor {
	return a
}

// ProvideTransactor provides ability to run DB transactions.
func ProvideTransactor(a AccessorTx) Transactor {
	return a
}
