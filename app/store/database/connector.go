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
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.ConnectorStore = (*connectorStore)(nil)

const (
	//nolint:goconst
	connectorQueryBase = `
		SELECT` + connectorColumns + `
		FROM connectors`

	connectorColumns = `
	connector_id,
	connector_description,
	connector_space_id,
	connector_uid,
	connector_data,
	connector_created,
	connector_updated,
	connector_version
	`
)

// NewConnectorStore returns a new ConnectorStore.
func NewConnectorStore(db *sqlx.DB) store.ConnectorStore {
	return &connectorStore{
		db: db,
	}
}

type connectorStore struct {
	db *sqlx.DB
}

// Find returns a connector given a connector ID.
func (s *connectorStore) Find(ctx context.Context, id int64) (*types.Connector, error) {
	const findQueryStmt = connectorQueryBase + `
		WHERE connector_id = $1`
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(types.Connector)
	if err := db.GetContext(ctx, dst, findQueryStmt, id); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to find connector")
	}
	return dst, nil
}

// FindByIdentifier returns a connector in a given space with a given identifier.
func (s *connectorStore) FindByIdentifier(
	ctx context.Context,
	spaceID int64,
	identifier string,
) (*types.Connector, error) {
	const findQueryStmt = connectorQueryBase + `
		WHERE connector_space_id = $1 AND connector_uid = $2`
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(types.Connector)
	if err := db.GetContext(ctx, dst, findQueryStmt, spaceID, identifier); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to find connector")
	}
	return dst, nil
}

// Create creates a connector.
func (s *connectorStore) Create(ctx context.Context, connector *types.Connector) error {
	const connectorInsertStmt = `
	INSERT INTO connectors (
		connector_description
		,connector_type
		,connector_space_id
		,connector_uid
		,connector_data
		,connector_created
		,connector_updated
		,connector_version
	) VALUES (
		:connector_description
		,:connector_type
		,:connector_space_id
		,:connector_uid
		,:connector_data
		,:connector_created
		,:connector_updated
		,:connector_version
	) RETURNING connector_id`
	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(connectorInsertStmt, connector)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to bind connector object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&connector.ID); err != nil {
		return database.ProcessSQLErrorf(err, "connector query failed")
	}

	return nil
}

func (s *connectorStore) Update(ctx context.Context, p *types.Connector) error {
	const connectorUpdateStmt = `
	UPDATE connectors
	SET
		connector_description = :connector_description
		,connector_uid = :connector_uid
		,connector_data = :connector_data
		,connector_type = :connector_type
		,connector_updated = :connector_updated
		,connector_version = :connector_version
	WHERE connector_id = :connector_id AND connector_version = :connector_version - 1`
	connector := *p

	connector.Version++
	connector.Updated = time.Now().UnixMilli()

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(connectorUpdateStmt, connector)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to bind connector object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to update connector")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	p.Version = connector.Version
	p.Updated = connector.Updated
	return nil
}

// UpdateOptLock updates the connector using the optimistic locking mechanism.
func (s *connectorStore) UpdateOptLock(ctx context.Context,
	connector *types.Connector,
	mutateFn func(connector *types.Connector) error,
) (*types.Connector, error) {
	for {
		dup := *connector

		err := mutateFn(&dup)
		if err != nil {
			return nil, err
		}

		err = s.Update(ctx, &dup)
		if err == nil {
			return &dup, nil
		}
		if !errors.Is(err, gitness_store.ErrVersionConflict) {
			return nil, err
		}

		connector, err = s.Find(ctx, connector.ID)
		if err != nil {
			return nil, err
		}
	}
}

// List lists all the connectors present in a space.
func (s *connectorStore) List(
	ctx context.Context,
	parentID int64,
	filter types.ListQueryFilter,
) ([]*types.Connector, error) {
	stmt := database.Builder.
		Select(connectorColumns).
		From("connectors").
		Where("connector_space_id = ?", fmt.Sprint(parentID))

	if filter.Query != "" {
		stmt = stmt.Where("LOWER(connector_uid) LIKE ?", fmt.Sprintf("%%%s%%", strings.ToLower(filter.Query)))
	}

	stmt = stmt.Limit(database.Limit(filter.Size))
	stmt = stmt.Offset(database.Offset(filter.Page, filter.Size))

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*types.Connector{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed executing custom list query")
	}

	return dst, nil
}

// Delete deletes a connector given a connector ID.
func (s *connectorStore) Delete(ctx context.Context, id int64) error {
	const connectorDeleteStmt = `
		DELETE FROM connectors
		WHERE connector_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, connectorDeleteStmt, id); err != nil {
		return database.ProcessSQLErrorf(err, "Could not delete connector")
	}

	return nil
}

// DeleteByIdentifier deletes a connector with a given identifier in a space.
func (s *connectorStore) DeleteByIdentifier(ctx context.Context, spaceID int64, identifier string) error {
	const connectorDeleteStmt = `
	DELETE FROM connectors
	WHERE connector_space_id = $1 AND connector_uid = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, connectorDeleteStmt, spaceID, identifier); err != nil {
		return database.ProcessSQLErrorf(err, "Could not delete connector")
	}

	return nil
}

// Count of connectors in a space.
func (s *connectorStore) Count(ctx context.Context, parentID int64, filter types.ListQueryFilter) (int64, error) {
	stmt := database.Builder.
		Select("count(*)").
		From("connectors").
		Where("connector_space_id = ?", parentID)

	if filter.Query != "" {
		stmt = stmt.Where("LOWER(connector_uid) LIKE ?", fmt.Sprintf("%%%s%%", filter.Query))
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(err, "Failed executing count query")
	}
	return count, nil
}
