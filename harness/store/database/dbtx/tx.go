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

import "database/sql"

// TxDefault represents default transaction options.
var TxDefault = &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false}

// TxDefaultReadOnly represents default transaction options for read-only transactions.
var TxDefaultReadOnly = &sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: true}

// TxSerializable represents serializable transaction options.
var TxSerializable = &sql.TxOptions{Isolation: sql.LevelSerializable, ReadOnly: false}
