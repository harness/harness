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
	"fmt"
	"time"

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

func (n NodeDao) GetByBlobIDAndRegistryID(ctx context.Context, blobID string, registryID int64) (*types.Node, error) {
	q := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(Nodes{}), ",")).
		From("nodes").
		Where("node_generic_blob_id = ? AND node_registry_id = ?", blobID, registryID).Limit(1)

	db := dbtx.GetAccessor(ctx, n.sqlDB)

	dst := new(Nodes)
	_sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.GetContext(ctx, dst, _sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find node with registry id %d", registryID)
	}

	return n.mapToNode(ctx, dst)
}

func (n NodeDao) FindByPathAndRegistryID(
	ctx context.Context, registryID int64, path string,
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

func (n NodeDao) CountByPathAndRegistryID(
	ctx context.Context, registryID int64, path string,
) (int64, error) {
	q := databaseg.Builder.
		Select("COUNT(*)").
		From("nodes").
		Where("node_is_file = true AND node_path LIKE ? AND node_registry_id = ?", path, registryID)

	db := dbtx.GetAccessor(ctx, n.sqlDB)

	sql, args, err := q.ToSql()
	if err != nil {
		return -1, errors.Wrap(err, "Failed to convert query to sql")
	}

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}
	return count, nil
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
	query, arg, err := db.BindNamed(sqlQuery, n.mapToInternalNode(node))
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

func (n NodeDao) DeleteByNodePathAndRegistryID(ctx context.Context, nodePath string, regID int64) (err error) {
	db := dbtx.GetAccessor(ctx, n.sqlDB)
	delStmt := databaseg.Builder.Delete("nodes").
		Where("(node_path = ? OR node_path LIKE ?)", nodePath, nodePath+"/%").
		Where("node_registry_id = ?", regID)

	delQuery, delArgs, err := delStmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to convert purge query to sql: %w", err)
	}

	_, err = db.ExecContext(ctx, delQuery, delArgs...)
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "the delete query failed")
	}

	return nil
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

func (n NodeDao) mapToInternalNode(node *types.Node) interface{} {
	if node.CreatedAt.IsZero() {
		node.CreatedAt = time.Now()
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

func (n NodeDao) GetFilesMetadataByPathAndRegistryID(
	ctx context.Context, registryID int64, path string,
	sortByField string,
	sortByOrder string,
	limit int,
	offset int,
	search string,
) (*[]types.FileNodeMetadata, error) {
	q := databaseg.Builder.
		Select(`n.node_name AS name,
		n.node_created_at AS created_at,
        n.node_path AS path,
		gb.generic_blob_sha_1  AS sha1,
		gb.generic_blob_sha_256  AS sha256,
		gb.generic_blob_sha_512  AS sha512,
        gb.generic_blob_md5   AS md5,
		 gb.generic_blob_size  AS size`).
		From("nodes n").
		Where("n.node_is_file = true").
		Join("generic_blobs gb ON gb.generic_blob_id = n.node_generic_blob_id").
		Where("n.node_is_file = true AND n.node_path LIKE ? AND n.node_registry_id = ?", path, registryID)

	db := dbtx.GetAccessor(ctx, n.sqlDB)

	q = q.OrderBy(sortByField + " " + sortByOrder).Limit(uint64(limit)).Offset(uint64(offset)) //nolint:gosec

	if search != "" {
		q = q.Where("name LIKE ?", sqlPartialMatch(search))
	}
	dst := []*FileNodeMetadataDB{}
	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find node with registry id %d", registryID)
	}

	return n.mapToNodesMetadata(dst)
}

func (n NodeDao) GetFileMetadataByPathAndRegistryID(
	ctx context.Context,
	registryID int64,
	path string,
) (*types.FileNodeMetadata, error) {
	q := databaseg.Builder.
		Select(`n.node_name AS name,
		n.node_created_at AS created_at,
        n.node_path AS path,
		gb.generic_blob_sha_1  AS sha1,
		gb.generic_blob_sha_256  AS sha256,
		gb.generic_blob_sha_512  AS sha512,
        gb.generic_blob_md5   AS md5,
		 gb.generic_blob_size  AS size`).
		From("nodes n").
		Where("n.node_is_file = true").
		Join("generic_blobs gb ON gb.generic_blob_id = n.node_generic_blob_id").
		Where("n.node_is_file = true AND n.node_path LIKE ? AND n.node_registry_id = ?", path, registryID)

	db := dbtx.GetAccessor(ctx, n.sqlDB)

	dst := FileNodeMetadataDB{}
	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.GetContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find node with registry id %d", registryID)
	}

	return n.mapToNodeMetadata(&dst), nil
}

func (n NodeDao) mapToNodesMetadata(dst []*FileNodeMetadataDB) (*[]types.FileNodeMetadata, error) {
	nodes := make([]types.FileNodeMetadata, 0, len(dst))
	for _, d := range dst {
		node := n.mapToNodeMetadata(d)
		nodes = append(nodes, *node)
	}
	return &nodes, nil
}

func (n NodeDao) mapToNodeMetadata(d *FileNodeMetadataDB) *types.FileNodeMetadata {
	return &types.FileNodeMetadata{
		Name:      d.Name,
		Path:      d.Path,
		Size:      d.Size,
		MD5:       d.MD5,
		Sha1:      d.Sha1,
		Sha256:    d.Sha256,
		Sha512:    d.Sha512,
		CreatedAt: d.CreatedAt,
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

type FileNodeMetadataDB struct {
	Name      string `db:"name"`
	CreatedAt int64  `db:"created_at"`
	Path      string `db:"path"`
	Sha1      string `db:"sha1"`
	Sha256    string `db:"sha256"`
	Sha512    string `db:"sha512"`
	MD5       string `db:"md5"`
	Size      int64  `db:"size"`
}
