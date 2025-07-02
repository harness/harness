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
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var _ store.GitSignatureResultStore = GitSignatureResultStore{}

// NewGitSignatureResultStore returns a new GitSignatureResultStore.
func NewGitSignatureResultStore(db *sqlx.DB) GitSignatureResultStore {
	return GitSignatureResultStore{
		db: db,
	}
}

// GitSignatureResultStore implements a store.GitSignatureResultStore backed by a relational database.
type GitSignatureResultStore struct {
	db *sqlx.DB
}

const (
	gitSignatureResultColumns = `
			 git_signature_result_repo_id
			,git_signature_result_object_sha
			,git_signature_result_object_time
			,git_signature_result_created
			,git_signature_result_updated
			,git_signature_result_result
			,git_signature_result_principal_id
			,git_signature_result_key_scheme
			,git_signature_result_key_id
			,git_signature_result_key_fingerprint`

	gitSignatureResultInsertQuery = `
		INSERT INTO git_signature_results (` + gitSignatureResultColumns + `
		) values (
			 :git_signature_result_repo_id
			,:git_signature_result_object_sha
			,:git_signature_result_object_time
			,:git_signature_result_created
			,:git_signature_result_updated
			,:git_signature_result_result
			,:git_signature_result_principal_id
			,:git_signature_result_key_scheme
			,:git_signature_result_key_id
			,:git_signature_result_key_fingerprint
		)`
)

type gitSignatureResult struct {
	RepoID         int64  `db:"git_signature_result_repo_id"`
	ObjectSHA      string `db:"git_signature_result_object_sha"`
	ObjectTime     int64  `db:"git_signature_result_object_time"`
	Created        int64  `db:"git_signature_result_created"`
	Updated        int64  `db:"git_signature_result_updated"`
	Result         string `db:"git_signature_result_result"`
	PrincipalID    int64  `db:"git_signature_result_principal_id"`
	KeyScheme      string `db:"git_signature_result_key_scheme"`
	KeyID          string `db:"git_signature_result_key_id"`
	KeyFingerprint string `db:"git_signature_result_key_fingerprint"`
}

func (s GitSignatureResultStore) Map(
	ctx context.Context,
	repoID int64,
	objectSHAs []sha.SHA,
) (map[sha.SHA]types.GitSignatureResult, error) {
	stmt := database.Builder.
		Select(gitSignatureResultColumns).
		From("git_signature_results").
		Where("git_signature_result_repo_id = ?", repoID).
		Where(squirrel.Eq{"git_signature_result_object_sha": objectSHAs})

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	sigVers := make([]gitSignatureResult, 0)
	if err = db.SelectContext(ctx, &sigVers, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err,
			"failed to execute list git signature verification results query")
	}

	sigVerMap := map[sha.SHA]types.GitSignatureResult{}
	for _, sigVer := range sigVers {
		o := mapToGitSignatureResult(sigVer)
		sigVerMap[o.ObjectSHA] = o
	}

	return sigVerMap, nil
}

func (s GitSignatureResultStore) Create(
	ctx context.Context,
	sigResult types.GitSignatureResult,
) error {
	db := dbtx.GetAccessor(ctx, s.db)

	sigResultInternal := mapToInternalGitSignatureResult(sigResult)

	query, arg, err := db.BindNamed(gitSignatureResultInsertQuery, &sigResultInternal)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err,
			"Failed to bind git signature verification result object")
	}

	if _, err = db.ExecContext(ctx, query, arg...); err != nil {
		return database.ProcessSQLErrorf(ctx, err,
			"Insert git signature verification result query failed")
	}

	return nil
}

func (s GitSignatureResultStore) TryCreateAll(
	ctx context.Context,
	sigResults []*types.GitSignatureResult,
) error {
	if len(sigResults) == 0 {
		return nil
	}

	db := dbtx.GetAccessor(ctx, s.db)

	const sql = gitSignatureResultInsertQuery + `
		ON CONFLICT DO NOTHING`

	stmt, err := db.PrepareNamedContext(ctx, sql)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err,
			"Failed to prepare git signature verification result statement")
	}

	defer stmt.Close()

	for _, sigResult := range sigResults {
		_, err = stmt.Exec(mapToInternalGitSignatureResult(*sigResult))
		if err != nil {
			return database.ProcessSQLErrorf(ctx, err,
				"Failed to insert git signature verification result")
		}
	}

	return nil
}

func (s GitSignatureResultStore) UpdateAll(
	ctx context.Context,
	result enum.GitSignatureResult,
	principalID int64,
	keyIDs, keyFingerprints []string,
) error {
	query := database.Builder.
		Update("git_signature_results").
		Set("git_signature_result_result", result).
		Set("git_signature_result_updated", time.Now().UnixMilli()).
		Where("git_signature_result_principal_id = ?", principalID)

	if len(keyIDs) > 0 {
		query = query.Where(squirrel.Eq{"git_signature_result_key_id": keyIDs})
	}
	if len(keyFingerprints) > 0 {
		query = query.Where(squirrel.Eq{"git_signature_result_key_fingerprint": keyFingerprints})
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	_, err = db.ExecContext(ctx, sql, args...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "failed to update git signatures")
	}

	return nil
}

func mapToInternalGitSignatureResult(sigVer types.GitSignatureResult) gitSignatureResult {
	return gitSignatureResult{
		RepoID:         sigVer.RepoID,
		ObjectSHA:      sigVer.ObjectSHA.String(),
		ObjectTime:     sigVer.ObjectTime,
		Created:        sigVer.Created,
		Updated:        sigVer.Updated,
		Result:         string(sigVer.Result),
		PrincipalID:    sigVer.PrincipalID,
		KeyScheme:      string(sigVer.KeyScheme),
		KeyID:          sigVer.KeyID,
		KeyFingerprint: sigVer.KeyFingerprint,
	}
}

func mapToGitSignatureResult(sigVer gitSignatureResult) types.GitSignatureResult {
	objectSHA, _ := sha.New(sigVer.ObjectSHA)
	return types.GitSignatureResult{
		RepoID:         sigVer.RepoID,
		ObjectSHA:      objectSHA,
		ObjectTime:     sigVer.ObjectTime,
		Created:        sigVer.Created,
		Updated:        sigVer.Updated,
		Result:         enum.GitSignatureResult(sigVer.Result),
		PrincipalID:    sigVer.PrincipalID,
		KeyScheme:      enum.PublicKeyScheme(sigVer.KeyScheme),
		KeyID:          sigVer.KeyID,
		KeyFingerprint: sigVer.KeyFingerprint,
	}
}
