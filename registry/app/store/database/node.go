//  Copyright 2023 Harness, Inc.
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
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/store/database/util"
	"github.com/harness/gitness/registry/types"
	store2 "github.com/harness/gitness/store"
	databaseg "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type NodeDao struct {
	sqlDB *sqlx.DB
}

func (n NodeDao) GetByPathAndRegistryID(ctx context.Context, registryID int64, path string) (*types.Node, error) {
	q := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(Nodes{}), ",")).
		From("nodes").
		Where("node_path = ? AND node_registry_id = ?", path, registryID)

	db := dbtx.GetAccessor(ctx, n.sqlDB)

	dst := new(Nodes)
	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find node with registry id %d", registryID)
	}

	return n.mapToNode(ctx, dst)
}

func (n NodeDao) Get(ctx context.Context, id int64) (*types.Node, error) {
	q := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(Nodes{}), ",")).
		From("nodes").
		Where("node_id = ?", id)

	db := dbtx.GetAccessor(ctx, n.sqlDB)

	dst := new(Nodes)
	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find node with id %d", id)
	}

	return n.mapToNode(ctx, dst)
}

func (n NodeDao) GetByNameAndRegistryID(ctx context.Context, registryID int64, name string) (*types.Node, error) {
	q := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(Nodes{}), ",")).
		From("nodes").
		Where("node_name = ? AND node_registry_id = ?", name, registryID)

	db := dbtx.GetAccessor(ctx, n.sqlDB)

	dst := new(Nodes)
	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find node with registry id %d", registryID)
	}

	return n.mapToNode(ctx, dst)
}

func (n NodeDao) FindByPathAndRegistryID(ctx context.Context, registryID int64, path string,
) (*types.Node, error) {
	q := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(Nodes{}), ",")).
		From("nodes").
		Where("node_path LIKE ? AND node_registry_id = ?", path, registryID).
		OrderBy("node_created_at DESC").Limit(1)

	db := dbtx.GetAccessor(ctx, n.sqlDB)

	dst := new(Nodes)
	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find node with registry id %d", registryID)
	}

	return n.mapToNode(ctx, dst)
}

func (n NodeDao) Create(ctx context.Context, node *types.Node) error {
	const sqlQuery = `
		INSERT INTO nodes ( 
		                   node_id,
		         node_name
				,node_registry_id
				,node_parent_id
				,node_is_file
				,node_path
				,node_generic_blob_id
				,node_created_at
				,node_created_by
		    ) VALUES (
		              :node_id,
		         :node_name
				,:node_registry_id
				,:node_parent_id
				,:node_is_file
				,:node_path
				,:node_generic_blob_id
				,:node_created_at
				,:node_created_by
		    )  ON CONFLICT (node_registry_id, node_path)
			DO UPDATE SET node_id = nodes.node_id,
			node_generic_blob_id = :node_generic_blob_id
		    RETURNING node_id`

	db := dbtx.GetAccessor(ctx, n.sqlDB)
	query, arg, err := db.BindNamed(sqlQuery, n.mapToInternalNode(ctx, node))
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to bind node object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&node.ID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, store2.ErrDuplicate) {
			return nil
		}
		return databaseg.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}
	return nil
}

func (n NodeDao) DeleteByID(_ context.Context, _ int64) (err error) {
	// TODO implement me
	panic("implement me")
}

func (n NodeDao) mapToNode(_ context.Context, dst *Nodes) (*types.Node, error) {
	var blobID, parentNodeID string
	if dst.BlobID != nil {
		blobID = *dst.BlobID // Dereference the pointer if it's not nil
	}

	if dst.ParentNodeID != nil {
		parentNodeID = *dst.ParentNodeID // Dereference the pointer if it's not nil
	}
	return &types.Node{
		ID:           dst.ID,
		Name:         dst.Name,
		RegistryID:   dst.RegistryID,
		IsFile:       dst.IsFile,
		NodePath:     dst.NodePath,
		BlobID:       blobID,
		ParentNodeID: parentNodeID,
		CreatedAt:    time.UnixMilli(dst.CreatedAt),
		CreatedBy:    dst.CreatedBy,
	}, nil
}

func (n NodeDao) mapToInternalNode(ctx context.Context, node *types.Node) interface{} {
	session, _ := request.AuthSessionFrom(ctx)

	if node.CreatedAt.IsZero() {
		node.CreatedAt = time.Now()
	}
	if node.CreatedBy == 0 {
		node.CreatedBy = session.Principal.ID
	}
	if node.ID == "" {
		node.ID = uuid.NewString()
	}

	var blobID, parentNodeID *string
	if node.BlobID != "" {
		blobID = &node.BlobID // Store the actual value of BlobID
	}

	if node.ParentNodeID != "" {
		parentNodeID = &node.ParentNodeID // Store the actual value of BlobID
	}

	return &Nodes{
		ID:           node.ID,
		Name:         node.Name,
		ParentNodeID: parentNodeID,
		RegistryID:   node.RegistryID,
		IsFile:       node.IsFile,
		NodePath:     node.NodePath,
		BlobID:       blobID,
		CreatedAt:    node.CreatedAt.UnixMilli(),
		CreatedBy:    node.CreatedBy,
	}
}

func NewNodeDao(sqlDB *sqlx.DB) store.NodesRepository {
	return &NodeDao{sqlDB: sqlDB}
}

type Nodes struct {
	ID           string  `db:"node_id"`
	Name         string  `db:"node_name"`
	RegistryID   int64   `db:"node_registry_id"`
	IsFile       bool    `db:"node_is_file"`
	NodePath     string  `db:"node_path"`
	BlobID       *string `db:"node_generic_blob_id"`
	ParentNodeID *string `db:"node_parent_id"`
	CreatedAt    int64   `db:"node_created_at"`
	CreatedBy    int64   `db:"node_created_by"`
}
