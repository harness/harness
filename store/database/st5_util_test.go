// Copyright 2026 Harness, Inc.
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
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestSt5SQLErrorClassification(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		unique     bool
		foreignKey bool
	}{
		{"sqlite unique", sqlite3.Error{ExtendedCode: sqlite3.ErrConstraintUnique}, true, false},
		{"sqlite primary key", sqlite3.Error{ExtendedCode: sqlite3.ErrConstraintPrimaryKey}, true, false},
		{"sqlite foreign key", sqlite3.Error{ExtendedCode: sqlite3.ErrConstraintForeignKey}, false, true},
		{"sqlite other", sqlite3.Error{ExtendedCode: sqlite3.ErrConstraintCheck}, false, false},
		{"postgres unique", &pq.Error{Code: "23505"}, true, false},
		{"postgres foreign key", &pq.Error{Code: pgerrcode.ForeignKeyViolation}, false, true},
		{"postgres other", &pq.Error{Code: pgerrcode.NotNullViolation}, false, false},
		{"wrapped", fmt.Errorf("wrapped: %w", &pq.Error{Code: "23505"}), true, false},
		{"plain", errors.New("plain"), false, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.unique, isSQLUniqueConstraintError(test.err))
			require.Equal(t, test.foreignKey, isSQLForeignKeyViolationError(test.err))
		})
	}
}

// st5FlakyConnector fails to connect a given number of times before it succeeds.
type st5FlakyConnector struct {
	failures int
	calls    int
}

var errSt5Connect = errors.New("st5 connect failed")

func (c *st5FlakyConnector) Connect(context.Context) (driver.Conn, error) {
	c.calls++
	if c.calls <= c.failures {
		return nil, errSt5Connect
	}

	return st5Conn{}, nil
}

func (c *st5FlakyConnector) Driver() driver.Driver { return nil }

type st5Conn struct{}

func (st5Conn) Prepare(string) (driver.Stmt, error) { return nil, errSt5Connect }
func (st5Conn) Close() error                        { return nil }
func (st5Conn) Begin() (driver.Tx, error)           { return nil, errSt5Connect }

func TestSt5PingDatabaseRetries(t *testing.T) {
	connector := &st5FlakyConnector{failures: 1}
	db := sqlx.NewDb(sql.OpenDB(connector), "st5")
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, pingDatabase(context.Background(), db))
	require.Equal(t, 2, connector.calls)
}
