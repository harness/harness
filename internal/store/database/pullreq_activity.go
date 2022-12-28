// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"encoding/json"
	"time"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.PullReqActivityStore = (*PullReqActivityStore)(nil)

// NewPullReqActivityStore returns a new PullReqJournalStore.
func NewPullReqActivityStore(db *sqlx.DB) *PullReqActivityStore {
	return &PullReqActivityStore{
		db: db,
	}
}

// PullReqActivityStore implements store.PullReqActivityStore backed by a relational database.
type PullReqActivityStore struct {
	db *sqlx.DB
}

// journal is used to fetch pull request data from the database.
// The object should be later re-packed into a different struct to return it as an API response.
type pullReqActivity struct {
	ID      int64 `db:"pullreq_activity_id"`
	Version int64 `db:"pullreq_activity_version"`

	CreatedBy int64    `db:"pullreq_activity_created_by"`
	Created   int64    `db:"pullreq_activity_created"`
	Updated   int64    `db:"pullreq_activity_updated"`
	Edited    int64    `db:"pullreq_activity_edited"`
	Deleted   null.Int `db:"pullreq_activity_deleted"`

	ParentID  null.Int `db:"pullreq_activity_parent_id"`
	RepoID    int64    `db:"pullreq_activity_repo_id"`
	PullReqID int64    `db:"pullreq_activity_pullreq_id"`

	Order    int64 `db:"pullreq_activity_order"`
	SubOrder int64 `db:"pullreq_activity_sub_order"`
	ReplySeq int64 `db:"pullreq_activity_reply_seq"`

	Type enum.PullReqActivityType `db:"pullreq_activity_type"`
	Kind enum.PullReqActivityKind `db:"pullreq_activity_kind"`

	Text     string          `db:"pullreq_activity_text"`
	Payload  json.RawMessage `db:"pullreq_activity_payload"`
	Metadata json.RawMessage `db:"pullreq_activity_metadata"`

	ResolvedBy null.Int `db:"pullreq_activity_resolved_by"`
	Resolved   null.Int `db:"pullreq_activity_resolved"`

	AuthorUID     string      `db:"author_uid"`
	AuthorName    string      `db:"author_name"`
	AuthorEmail   string      `db:"author_email"`
	ResolverUID   null.String `db:"resolver_uid"`
	ResolverName  null.String `db:"resolver_name"`
	ResolverEmail null.String `db:"resolver_email"`
}

const (
	pullreqActivityColumns = `
		 pullreq_activity_id
		,pullreq_activity_version
		,pullreq_activity_created_by
		,pullreq_activity_created
		,pullreq_activity_updated
		,pullreq_activity_edited
		,pullreq_activity_deleted
		,pullreq_activity_parent_id
		,pullreq_activity_repo_id
		,pullreq_activity_pullreq_id
		,pullreq_activity_order
		,pullreq_activity_sub_order
		,pullreq_activity_reply_seq
		,pullreq_activity_type
		,pullreq_activity_kind
		,pullreq_activity_text
		,pullreq_activity_payload
		,pullreq_activity_metadata
		,pullreq_activity_resolved_by
		,pullreq_activity_resolved
		,author.principal_uid as "author_uid"
		,author.principal_displayName as "author_name"
		,author.principal_email as "author_email"
		,resolver.principal_uid as "resolver_uid"
		,resolver.principal_displayName as "resolver_name"
		,resolver.principal_email as "resolver_email"`

	pullreqActivitySelectBase = `
	SELECT` + pullreqActivityColumns + `
	FROM pullreq_activities
	INNER JOIN principals author on author.principal_id = pullreq_activity_created_by
	LEFT JOIN principals resolver on resolver.principal_id = pullreq_activity_resolved_by`
)

// Find finds the pull request activity by id.
func (s *PullReqActivityStore) Find(ctx context.Context, id int64) (*types.PullReqActivity, error) {
	const sqlQuery = pullreqActivitySelectBase + `
	WHERE pullreq_activity_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &pullReqActivity{}
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, processSQLErrorf(err, "Select query failed")
	}

	return mapPullReqActivity(dst), nil
}

// Create creates a new pull request.
func (s *PullReqActivityStore) Create(ctx context.Context, act *types.PullReqActivity) error {
	const sqlQuery = `
	INSERT INTO pullreq_activities (
		 pullreq_activity_version
		,pullreq_activity_created_by
		,pullreq_activity_created
		,pullreq_activity_updated
		,pullreq_activity_edited
		,pullreq_activity_deleted
		,pullreq_activity_parent_id
		,pullreq_activity_repo_id
		,pullreq_activity_pullreq_id
		,pullreq_activity_order
		,pullreq_activity_sub_order
		,pullreq_activity_reply_seq
		,pullreq_activity_type
		,pullreq_activity_kind
		,pullreq_activity_text
		,pullreq_activity_payload
		,pullreq_activity_metadata
		,pullreq_activity_resolved_by
		,pullreq_activity_resolved
	) values (
		 :pullreq_activity_version
		,:pullreq_activity_created_by
		,:pullreq_activity_created
		,:pullreq_activity_updated
		,:pullreq_activity_edited
		,:pullreq_activity_deleted
		,:pullreq_activity_parent_id
		,:pullreq_activity_repo_id
		,:pullreq_activity_pullreq_id
		,:pullreq_activity_order
		,:pullreq_activity_sub_order
		,:pullreq_activity_reply_seq
		,:pullreq_activity_type
		,:pullreq_activity_kind
		,:pullreq_activity_text
		,:pullreq_activity_payload
		,:pullreq_activity_metadata
		,:pullreq_activity_resolved_by
		,:pullreq_activity_resolved
	) RETURNING pullreq_activity_id`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, mapInternalPullReqActivity(act))
	if err != nil {
		return processSQLErrorf(err, "Failed to bind pull request activity object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&act.ID); err != nil {
		return processSQLErrorf(err, "Failed to insert pull request activity")
	}

	return nil
}

// Update updates the pull request.
func (s *PullReqActivityStore) Update(ctx context.Context, act *types.PullReqActivity) error {
	const sqlQuery = `
	UPDATE pullreq_activities
	SET
	     pullreq_activity_version = :pullreq_activity_version
		,pullreq_activity_updated = :pullreq_activity_updated
		,pullreq_activity_edited = :pullreq_activity_edited
		,pullreq_activity_deleted = :pullreq_activity_deleted
		,pullreq_activity_reply_seq = :pullreq_activity_reply_seq
		,pullreq_activity_text = :pullreq_activity_text
		,pullreq_activity_payload = :pullreq_activity_payload
		,pullreq_activity_metadata = :pullreq_activity_metadata
		,pullreq_activity_resolved_by = :pullreq_activity_resolved_by
		,pullreq_activity_resolved = :pullreq_activity_resolved
	WHERE pullreq_activity_id = :pullreq_activity_id AND pullreq_activity_version = :pullreq_activity_version - 1`

	db := dbtx.GetAccessor(ctx, s.db)

	updatedAt := time.Now()

	dbAct := mapInternalPullReqActivity(act)
	dbAct.Version++
	dbAct.Updated = updatedAt.UnixMilli()

	query, arg, err := db.BindNamed(sqlQuery, dbAct)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind pull request activity object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return processSQLErrorf(err, "Failed to update pull request activity")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return processSQLErrorf(err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return store.ErrConflict
	}

	act.Version = dbAct.Version
	act.Updated = dbAct.Updated

	return nil
}

// UpdateReplySeq updates the pull request activity's reply sequence.
func (s *PullReqActivityStore) UpdateReplySeq(ctx context.Context,
	act *types.PullReqActivity) (*types.PullReqActivity, error) {
	for {
		dup := *act

		dup.ReplySeq++
		err := s.Update(ctx, &dup)
		if err == nil {
			return &dup, nil
		}
		if !errors.Is(err, store.ErrConflict) {
			return nil, err
		}

		act, err = s.Find(ctx, act.ID)
		if err != nil {
			return nil, err
		}
	}
}

// Count of pull requests for a repo.
func (s *PullReqActivityStore) Count(ctx context.Context, prID int64,
	opts *types.PullReqActivityFilter) (int64, error) {
	stmt := builder.
		Select("count(*)").
		From("pullreq_activities").
		Where("pullreq_activity_pullreq_id = ?", prID)

	if len(opts.Types) == 1 {
		stmt = stmt.Where("pullreq_activity_type = ?", opts.Types[0])
	} else if len(opts.Types) > 1 {
		stmt = stmt.Where(squirrel.Eq{"pullreq_activity_type": opts.Types})
	}

	if len(opts.Kinds) == 1 {
		stmt = stmt.Where("pullreq_activity_kind = ?", opts.Kinds[0])
	} else if len(opts.Kinds) > 1 {
		stmt = stmt.Where(squirrel.Eq{"pullreq_activity_kind": opts.Kinds})
	}

	if opts.Since != 0 {
		stmt = stmt.Where("pullreq_created >= ?", opts.Since)
	}

	if opts.Until != 0 {
		stmt = stmt.Where("pullreq_created < ?", opts.Until)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, processSQLErrorf(err, "Failed executing count query")
	}

	return count, nil
}

// List returns a list of pull requests for a repo.
func (s *PullReqActivityStore) List(ctx context.Context, prID int64,
	opts *types.PullReqActivityFilter) ([]*types.PullReqActivity, error) {
	stmt := builder.
		Select(pullreqActivityColumns).
		From("pullreq_activities").
		InnerJoin("principals author on author.principal_id = pullreq_activity_created_by").
		LeftJoin("principals resolver on resolver.principal_id = pullreq_activity_resolved_by").
		Where("pullreq_activity_pullreq_id = ?", prID)

	if len(opts.Types) == 1 {
		stmt = stmt.Where("pullreq_activity_type = ?", opts.Types[0])
	} else if len(opts.Types) > 1 {
		stmt = stmt.Where(squirrel.Eq{"pullreq_activity_type": opts.Types})
	}

	if len(opts.Kinds) == 1 {
		stmt = stmt.Where("pullreq_activity_kind = ?", opts.Kinds[0])
	} else if len(opts.Kinds) > 1 {
		stmt = stmt.Where(squirrel.Eq{"pullreq_activity_kind": opts.Kinds})
	}

	if opts.Since != 0 {
		stmt = stmt.Where("pullreq_created >= ?", opts.Since)
	}

	if opts.Until != 0 {
		stmt = stmt.Where("pullreq_created < ?", opts.Until)
	}

	if opts.Limit > 0 {
		stmt = stmt.Limit(uint64(limit(opts.Limit)))
	}

	stmt = stmt.OrderBy("pullreq_activity_order asc", "pullreq_activity_sub_order asc")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert pull request activity query to sql")
	}

	dst := make([]*pullReqActivity, 0)

	db := dbtx.GetAccessor(ctx, s.db)

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, processSQLErrorf(err, "Failed executing pull request activity list query")
	}

	return mapSlicePullReqActivity(dst), nil
}

func mapPullReqActivity(act *pullReqActivity) *types.PullReqActivity {
	m := &types.PullReqActivity{
		ID:         act.ID,
		Version:    act.Version,
		CreatedBy:  act.CreatedBy,
		Created:    act.Created,
		Updated:    act.Updated,
		Edited:     act.Edited,
		Deleted:    act.Deleted.Ptr(),
		ParentID:   act.ParentID.Ptr(),
		RepoID:     act.RepoID,
		PullReqID:  act.PullReqID,
		Order:      act.Order,
		SubOrder:   act.SubOrder,
		ReplySeq:   act.ReplySeq,
		Type:       act.Type,
		Kind:       act.Kind,
		Text:       act.Text,
		Payload:    make(map[string]interface{}),
		Metadata:   make(map[string]interface{}),
		ResolvedBy: act.ResolvedBy.Ptr(),
		Resolved:   act.Resolved.Ptr(),
		Author:     types.PrincipalInfo{},
		Resolver:   nil,
	}
	m.Author = types.PrincipalInfo{
		ID:    act.CreatedBy,
		UID:   act.AuthorUID,
		Name:  act.AuthorName,
		Email: act.AuthorEmail,
	}

	_ = json.Unmarshal(act.Payload, &m.Payload)
	_ = json.Unmarshal(act.Metadata, &m.Metadata)

	if act.ResolvedBy.Valid {
		m.Resolver = &types.PrincipalInfo{
			ID:    act.ResolvedBy.Int64,
			UID:   act.ResolverUID.String,
			Name:  act.ResolverName.String,
			Email: act.ResolverEmail.String,
		}
	}

	return m
}

func mapInternalPullReqActivity(act *types.PullReqActivity) *pullReqActivity {
	m := &pullReqActivity{
		ID:         act.ID,
		Version:    act.Version,
		CreatedBy:  act.CreatedBy,
		Created:    act.Created,
		Updated:    act.Updated,
		Edited:     act.Edited,
		Deleted:    null.IntFromPtr(act.Deleted),
		ParentID:   null.IntFromPtr(act.ParentID),
		RepoID:     act.RepoID,
		PullReqID:  act.PullReqID,
		Order:      act.Order,
		SubOrder:   act.SubOrder,
		ReplySeq:   act.ReplySeq,
		Type:       act.Type,
		Kind:       act.Kind,
		Text:       act.Text,
		Payload:    nil,
		Metadata:   nil,
		ResolvedBy: null.IntFromPtr(act.ResolvedBy),
		Resolved:   null.IntFromPtr(act.Resolved),
	}

	m.Payload, _ = json.Marshal(act.Payload)
	m.Metadata, _ = json.Marshal(act.Metadata)

	return m
}

func mapSlicePullReqActivity(a []*pullReqActivity) []*types.PullReqActivity {
	m := make([]*types.PullReqActivity, len(a))
	for i, act := range a {
		m[i] = mapPullReqActivity(act)
	}
	return m
}
