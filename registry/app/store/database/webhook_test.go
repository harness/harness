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
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/harness/gitness/registry/app/store"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordDriver struct {
	lastQuery string
}

func (d *recordDriver) Open(string) (driver.Conn, error) { return &recordConn{d: d}, nil }

type recordConn struct {
	d *recordDriver
}

func (c *recordConn) Prepare(q string) (driver.Stmt, error) {
	c.d.lastQuery = q
	return nil, errors.New("captured")
}
func (c *recordConn) Close() error              { return nil }
func (c *recordConn) Begin() (driver.Tx, error) { return nil, errors.New("no tx") }

var (
	recorderDriverOnce sync.Once
	globalRecordDriver = &recordDriver{}
)

func getTestWebhookRepo(t *testing.T) (store.WebhooksRepository, *recordDriver) {
	recorderDriverOnce.Do(func() {
		sql.Register("recorder-webhook-test", globalRecordDriver)
	})
	sdb, err := sql.Open("recorder-webhook-test", "unused")
	require.NoError(t, err)
	return NewWebhookDao(sqlx.NewDb(sdb, "postgres")), globalRecordDriver
}

func TestWebhookDao_ListByRegistry_SQLInjectionPrevention(t *testing.T) {
	repo, rec := getTestWebhookRepo(t)
	payload := "ASC,(SELECT CASE WHEN (substr((select principal_salt from principals limit 1),1,1)='a')" +
		" THEN registry_webhook_name ELSE registry_webhook_id END)"

	_, _ = repo.ListByRegistry(context.Background(), "name", payload, 10, 0, "", 1)
	sent := rec.lastQuery

	for _, needle := range []string{"SELECT CASE WHEN", "principal_salt", "registry_webhook_id END"} {
		assert.False(t, strings.Contains(sent, needle), "SQL should not contain payload needle: %s", needle)
	}
	assert.Contains(t, sent, "ORDER BY registry_webhook_name ASC")
}

func TestWebhookDao_ListByRegistry_SortOrder(t *testing.T) {
	tests := []struct {
		name          string
		sortByField   string
		sortByOrder   string
		expectedOrder string
	}{
		{
			name:          "valid_desc",
			sortByField:   "name",
			sortByOrder:   "DESC",
			expectedOrder: "ORDER BY registry_webhook_name DESC",
		},
		{
			name:          "lowercase_desc",
			sortByField:   "name",
			sortByOrder:   "desc",
			expectedOrder: "ORDER BY registry_webhook_name DESC",
		},
		{
			name:          "valid_asc",
			sortByField:   "name",
			sortByOrder:   "ASC",
			expectedOrder: "ORDER BY registry_webhook_name ASC",
		},
		{
			name:          "empty_order_defaults_to_asc",
			sortByField:   "name",
			sortByOrder:   "",
			expectedOrder: "ORDER BY registry_webhook_name ASC",
		},
		{
			name:          "invalid_order_defaults_to_asc",
			sortByField:   "name",
			sortByOrder:   "INVALID_ORDER_INJECTION",
			expectedOrder: "ORDER BY registry_webhook_name ASC",
		},
	}

	repo, rec := getTestWebhookRepo(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _ = repo.ListByRegistry(context.Background(), tt.sortByField, tt.sortByOrder, 10, 0, "", 1)
			assert.Contains(t, rec.lastQuery, tt.expectedOrder)
		})
	}
}
