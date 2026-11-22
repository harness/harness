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
	"strings"
	"sync"
	"testing"

	"github.com/harness/gitness/registry/app/store"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	recorderDriverUpstreamOnce sync.Once
	globalUpstreamRecordDriver = &recordDriver{}
)

func getTestUpstreamProxyRepo(t *testing.T) (store.UpstreamProxyConfigRepository, *recordDriver) {
	recorderDriverUpstreamOnce.Do(func() {
		sql.Register("recorder-upstream-test", globalUpstreamRecordDriver)
	})
	sdb, err := sql.Open("recorder-upstream-test", "unused")
	require.NoError(t, err)
	return &UpstreamproxyDao{db: sqlx.NewDb(sdb, "postgres")}, globalUpstreamRecordDriver
}

func TestUpstreamProxyDao_GetAll_SQLInjectionPrevention(t *testing.T) {
	repo, rec := getTestUpstreamProxyRepo(t)
	payload := "ASC,(SELECT CASE WHEN (1=1) THEN 1 ELSE 0 END)"

	_, _ = repo.GetAll(context.Background(), 1, nil, "name", payload, 10, 0, "")
	sent := rec.lastQuery

	assert.False(t, strings.Contains(sent, "SELECT CASE WHEN"), "SQL should not contain payload")
	assert.Contains(t, sent, "ORDER BY r.registry_name ASC")
}

func TestUpstreamProxyDao_GetAll_SortOrderAndField(t *testing.T) {
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
			expectedOrder: "ORDER BY r.registry_name DESC",
		},
		{
			name:          "lowercase_desc",
			sortByField:   "name",
			sortByOrder:   "desc",
			expectedOrder: "ORDER BY r.registry_name DESC",
		},
		{
			name:          "created_at_asc",
			sortByField:   "created_at",
			sortByOrder:   "ASC",
			expectedOrder: "ORDER BY r.registry_created_at ASC",
		},
		{
			name:          "invalid_field_defaults_to_name",
			sortByField:   "invalid_field; DROP TABLE registries;--",
			sortByOrder:   "ASC",
			expectedOrder: "ORDER BY r.registry_name ASC",
		},
		{
			name:          "invalid_order_defaults_to_asc",
			sortByField:   "updated_at",
			sortByOrder:   "MALICIOUS_ORDER",
			expectedOrder: "ORDER BY r.registry_updated_at ASC",
		},
	}

	repo, rec := getTestUpstreamProxyRepo(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _ = repo.GetAll(context.Background(), 1, nil, tt.sortByField, tt.sortByOrder, 10, 0, "")
			assert.Contains(t, rec.lastQuery, tt.expectedOrder)
		})
	}
}
