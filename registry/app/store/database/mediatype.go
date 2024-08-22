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

	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/jmoiron/sqlx"
	errors2 "github.com/pkg/errors"
)

type mediaTypesDao struct {
	db *sqlx.DB
}

func NewMediaTypesDao(db *sqlx.DB) store.MediaTypesRepository {
	return &mediaTypesDao{
		db: db,
	}
}

func (mt mediaTypesDao) MediaTypeExists(ctx context.Context, mediaType string) (bool, error) {
	stmt := database.Builder.Select("EXISTS (SELECT 1 FROM media_types WHERE mt_media_type = ?)")
	sql, args, err := stmt.ToSql()
	if err != nil {
		return false, errors2.Wrap(err, "Failed to convert query to sql")
	}
	args = append(args, mediaType)

	var exists bool
	db := dbtx.GetAccessor(ctx, mt.db)

	if err = db.GetContext(ctx, &exists, sql, args...); err != nil {
		return false, database.ProcessSQLErrorf(ctx, err, "Failed to check if media type exists")
	}

	return exists, nil
}

func (mt mediaTypesDao) MapMediaType(ctx context.Context, mediaType string) (int64, error) {
	stmt := database.Builder.Select("mt_id").
		From("media_types").
		Where("mt_media_type = ?", mediaType)

	db := dbtx.GetAccessor(ctx, mt.db)
	var id int64
	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors2.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.GetContext(ctx, &id, sql, args...); err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed to find repo")
	}

	return id, nil
}
