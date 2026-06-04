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
	"time"

	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
)

var _ store.LinkedPullReqStore = (*LinkedPullReqStore)(nil)

// NewLinkedPullReqStore returns a new LinkedPullReqStore.
func NewLinkedPullReqStore(db *sqlx.DB) *LinkedPullReqStore {
	return &LinkedPullReqStore{db: db}
}

// LinkedPullReqStore implements store.LinkedPullReqStore backed by a relational database.
type LinkedPullReqStore struct {
	db *sqlx.DB
}

type linkedPullReq struct {
	PullReqID int64 `db:"linked_pullreq_id"`

	ProviderType   string `db:"linked_pullreq_provider_type"`
	ProviderRepoID string `db:"linked_pullreq_provider_repo_id"`
	ProviderURL    string `db:"linked_pullreq_provider_url"`

	ProviderAuthorLogin     string `db:"linked_pullreq_provider_author_login"`
	ProviderAuthorAvatarURL string `db:"linked_pullreq_provider_author_avatar_url"`
	ProviderAuthorURL       string `db:"linked_pullreq_provider_author_url"`

	ProviderUpdatedAt int64 `db:"linked_pullreq_provider_updated_at"`

	MergerLogin string `db:"linked_pullreq_merger_login"`

	LastSyncedAt int64 `db:"linked_pullreq_last_synced_at"`
}

const (
	linkedPullReqColumns = `
		 linked_pullreq_id
		,linked_pullreq_provider_type
		,linked_pullreq_provider_repo_id
		,linked_pullreq_provider_url
		,linked_pullreq_provider_author_login
		,linked_pullreq_provider_author_avatar_url
		,linked_pullreq_provider_author_url
		,linked_pullreq_provider_updated_at
		,linked_pullreq_merger_login
		,linked_pullreq_last_synced_at`

	linkedPullReqSelectBase = `
	SELECT` + linkedPullReqColumns + `
	FROM linked_pullreqs`
)

func (s *LinkedPullReqStore) Find(ctx context.Context, pullReqID int64) (*types.LinkedPullReq, error) {
	const sqlQuery = linkedPullReqSelectBase + `
	WHERE linked_pullreq_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &linkedPullReq{}
	if err := db.GetContext(ctx, dst, sqlQuery, pullReqID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find linked pull request")
	}

	return mapToLinkedPullReq(dst), nil
}

// FindByLinkedRepoAndProviderPR joins through pullreqs to scope the lookup
// to a single linked repo and match on the upstream PR number stored on
// the parent pullreq row.
func (s *LinkedPullReqStore) FindByLinkedRepoAndProviderPR(
	ctx context.Context,
	linkedRepoID int64,
	providerType string,
	providerRepoID string,
	providerPRNumber int,
) (*types.LinkedPullReq, error) {
	const sqlQuery = `
	SELECT` + linkedPullReqColumns + `
	FROM linked_pullreqs
	INNER JOIN pullreqs ON pullreq_id = linked_pullreq_id
	WHERE linked_pullreq_provider_type = $1
	  AND linked_pullreq_provider_repo_id = $2
	  AND pullreq_target_repo_id = $3
	  AND pullreq_number = $4
	LIMIT 1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &linkedPullReq{}
	if err := db.GetContext(ctx, dst, sqlQuery,
		providerType, providerRepoID, linkedRepoID, providerPRNumber); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err,
			"Failed to find linked pull request by linked repo and provider PR")
	}

	return mapToLinkedPullReq(dst), nil
}

// Create inserts a linked-PR row. The parent pullreqs row must already exist.
func (s *LinkedPullReqStore) Create(ctx context.Context, v *types.LinkedPullReq) error {
	const sqlQuery = `
	INSERT INTO linked_pullreqs (
		 linked_pullreq_id
		,linked_pullreq_provider_type
		,linked_pullreq_provider_repo_id
		,linked_pullreq_provider_url
		,linked_pullreq_provider_author_login
		,linked_pullreq_provider_author_avatar_url
		,linked_pullreq_provider_author_url
		,linked_pullreq_provider_updated_at
		,linked_pullreq_merger_login
		,linked_pullreq_last_synced_at
	) values (
		 :linked_pullreq_id
		,:linked_pullreq_provider_type
		,:linked_pullreq_provider_repo_id
		,:linked_pullreq_provider_url
		,:linked_pullreq_provider_author_login
		,:linked_pullreq_provider_author_avatar_url
		,:linked_pullreq_provider_author_url
		,:linked_pullreq_provider_updated_at
		,:linked_pullreq_merger_login
		,:linked_pullreq_last_synced_at
	)`

	row := mapToInternalLinkedPullReq(v)
	if row.LastSyncedAt == 0 {
		row.LastSyncedAt = time.Now().UnixMilli()
	}

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, row)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind linked pull request object")
	}

	if _, err = db.ExecContext(ctx, query, arg...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to insert linked pull request")
	}

	v.LastSyncedAt = row.LastSyncedAt
	return nil
}

// Update overwrites the mutable provider-side metadata. PullReqID, provider_type
// and provider_repo_id are immutable and excluded from the SET clause.
func (s *LinkedPullReqStore) Update(ctx context.Context, v *types.LinkedPullReq) error {
	const sqlQuery = `
	UPDATE linked_pullreqs
	SET
		 linked_pullreq_provider_url = :linked_pullreq_provider_url
		,linked_pullreq_provider_author_login = :linked_pullreq_provider_author_login
		,linked_pullreq_provider_author_avatar_url = :linked_pullreq_provider_author_avatar_url
		,linked_pullreq_provider_author_url = :linked_pullreq_provider_author_url
		,linked_pullreq_provider_updated_at = :linked_pullreq_provider_updated_at
		,linked_pullreq_merger_login = :linked_pullreq_merger_login
		,linked_pullreq_last_synced_at = :linked_pullreq_last_synced_at
	WHERE linked_pullreq_id = :linked_pullreq_id`

	row := mapToInternalLinkedPullReq(v)
	row.LastSyncedAt = time.Now().UnixMilli()

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, row)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind linked pull request object")
	}

	res, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update linked pull request")
	}
	// Zero rows usually means the parent pullreqs row was CASCADE-deleted
	// between FindByLinkedRepoAndProviderPR and Update.
	if n, _ := res.RowsAffected(); n == 0 {
		return gitness_store.ErrResourceNotFound
	}

	v.LastSyncedAt = row.LastSyncedAt
	return nil
}

func mapToLinkedPullReq(src *linkedPullReq) *types.LinkedPullReq {
	return &types.LinkedPullReq{
		PullReqID:               src.PullReqID,
		ProviderType:            src.ProviderType,
		ProviderRepoID:          src.ProviderRepoID,
		ProviderURL:             src.ProviderURL,
		ProviderAuthorLogin:     src.ProviderAuthorLogin,
		ProviderAuthorAvatarURL: src.ProviderAuthorAvatarURL,
		ProviderAuthorURL:       src.ProviderAuthorURL,
		ProviderUpdatedAt:       src.ProviderUpdatedAt,
		MergerLogin:             src.MergerLogin,
		LastSyncedAt:            src.LastSyncedAt,
	}
}

func mapToInternalLinkedPullReq(src *types.LinkedPullReq) *linkedPullReq {
	return &linkedPullReq{
		PullReqID:               src.PullReqID,
		ProviderType:            src.ProviderType,
		ProviderRepoID:          src.ProviderRepoID,
		ProviderURL:             src.ProviderURL,
		ProviderAuthorLogin:     src.ProviderAuthorLogin,
		ProviderAuthorAvatarURL: src.ProviderAuthorAvatarURL,
		ProviderAuthorURL:       src.ProviderAuthorURL,
		ProviderUpdatedAt:       src.ProviderUpdatedAt,
		MergerLogin:             src.MergerLogin,
		LastSyncedAt:            src.LastSyncedAt,
	}
}
