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

	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
)

type repoLang struct {
	RepoID   int64  `db:"repo_lang_repo_id"`
	Language string `db:"repo_lang_language"`
	Bytes    int64  `db:"repo_lang_bytes"`
	Files    int64  `db:"repo_lang_files"`
}

type RepoLangStore struct {
	db *sqlx.DB
}

func NewRepoLangStore(db *sqlx.DB) *RepoLangStore {
	return &RepoLangStore{db: db}
}

func (s *RepoLangStore) InsertByRepoID(
	ctx context.Context,
	repoID int64,
	langs []*types.RepoLangStat,
) error {
	if len(langs) == 0 {
		return nil
	}

	db := dbtx.GetAccessor(ctx, s.db)

	const insertSQL = `
		INSERT INTO repo_languages (
			repo_lang_repo_id,
			repo_lang_language,
			repo_lang_bytes,
			repo_lang_files
		) VALUES (
			:repo_lang_repo_id,
			:repo_lang_language,
			:repo_lang_bytes,
			:repo_lang_files
		)`

	rows := mapToInternalRepoLangs(repoID, langs)

	_, err := sqlx.NamedExecContext(ctx, db, insertSQL, rows)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed executing insert many query")
	}

	return nil
}

func (s *RepoLangStore) DeleteByRepoID(
	ctx context.Context,
	repoID int64,
) error {
	db := dbtx.GetAccessor(ctx, s.db)

	_, err := db.ExecContext(
		ctx,
		`DELETE FROM repo_languages WHERE repo_lang_repo_id = $1`,
		repoID,
	)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed executing delete query")
	}

	return nil
}

func (s *RepoLangStore) ListByRepoID(
	ctx context.Context,
	repoID int64,
) ([]*types.RepoLangStat, error) {
	db := dbtx.GetAccessor(ctx, s.db)

	const query = `
		SELECT
			repo_lang_language,
			repo_lang_bytes,
			repo_lang_files
		FROM repo_languages
		WHERE repo_lang_repo_id = $1
		ORDER BY repo_lang_bytes DESC, repo_lang_language ASC`

	var rows []*repoLang
	if err := db.SelectContext(ctx, &rows, query, repoID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing find query")
	}

	return mapToRepoLangStats(rows), nil
}

func mapToInternalRepoLangs(
	repoID int64,
	stats []*types.RepoLangStat,
) []*repoLang {
	rows := make([]*repoLang, 0, len(stats))

	for _, s := range stats {
		rows = append(rows, &repoLang{
			RepoID:   repoID,
			Language: s.Language,
			Bytes:    s.Bytes,
			Files:    s.Files,
		})
	}

	return rows
}

func mapToRepoLangStats(
	rows []*repoLang,
) []*types.RepoLangStat {
	res := make([]*types.RepoLangStat, len(rows))

	for i, r := range rows {
		res[i] = &types.RepoLangStat{
			Language: r.Language,
			Bytes:    r.Bytes,
			Files:    r.Files,
		}
	}

	return res
}
