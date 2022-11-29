// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package dbtx

import "database/sql"

// TxDefault represents default transaction options
var TxDefault = &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false}

// TxDefaultReadOnly represents default transaction options for read-only transactions
var TxDefaultReadOnly = &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: true}

// TxSerializable represents serializable transaction options
var TxSerializable = &sql.TxOptions{Isolation: sql.LevelSerializable, ReadOnly: false}
