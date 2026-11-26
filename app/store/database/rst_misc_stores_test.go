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

package database_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/job"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestRstEncodeToSQLXJSON(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"nil", nil, "null"},
		{"map", map[string]string{"a": "b"}, `{"a":"b"}`},
		{"slice", []string{"x", "y"}, `["x","y"]`},
		{"string", "text", `"text"`},
		// an unsupported value is absorbed and results in empty JSON text.
		{"unsupported", make(chan int), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, string(database.EncodeToSQLXJSON(tt.in)))
		})
	}
}

func TestRstPartialMatch(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantExpr  string
		wantValue string
	}{
		{"plain", "abc", "LOWER(col) LIKE '%' || LOWER(?) || '%'", "abc"},
		{"underscore", "a_b", `LOWER(col) LIKE '%' || LOWER(?) || '%' ESCAPE '\'`, `a\_b`},
		{"percent", "a%b", `LOWER(col) LIKE '%' || LOWER(?) || '%' ESCAPE '\'`, `a\%b`},
		{"backslash", `a\b`, `LOWER(col) LIKE '%' || LOWER(?) || '%' ESCAPE '\'`, `a\\b`},
		{"all", `\_%`, `LOWER(col) LIKE '%' || LOWER(?) || '%' ESCAPE '\'`, `\\\_\%`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, value := database.PartialMatch("col", tt.value)
			require.Equal(t, tt.wantExpr, expr)
			require.Equal(t, tt.wantValue, value)
		})
	}
}

func TestRstPluginStore(t *testing.T) {
	db, teardown := setupDB(t)
	t.Cleanup(teardown)
	ctx := context.Background()
	plugins := database.NewPluginStore(db)

	mk := func(uid, version string) *types.Plugin {
		p := &types.Plugin{
			Identifier: uid, Description: "desc " + uid, Type: "step", Version: version, Logo: "logo.png",
			Spec: "spec: " + uid,
		}
		require.NoError(t, plugins.Create(ctx, p))

		return p
	}

	docker := mk("docker", "1.0.0")
	mk("docker-compose", "2.0.0")
	mk("slack", "1.0.0")

	t.Run("duplicate", func(t *testing.T) {
		err := plugins.Create(ctx, &types.Plugin{Identifier: "docker", Version: "1.0.0"})
		require.ErrorIs(t, err, gitness_store.ErrDuplicate)
	})

	t.Run("find", func(t *testing.T) {
		got, err := plugins.Find(ctx, "docker-compose", "2.0.0")
		require.NoError(t, err)
		require.Equal(t, "spec: docker-compose", got.Spec)
		require.Equal(t, "logo.png", got.Logo)

		_, err = plugins.Find(ctx, "docker", "2.0.0")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("list and count", func(t *testing.T) {
		got, err := plugins.ListAll(ctx)
		require.NoError(t, err)
		require.Len(t, got, 3)

		got, err = plugins.List(ctx, types.ListQueryFilter{Query: "SLA"})
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, "slack", got[0].Identifier)

		got, err = plugins.List(ctx, types.ListQueryFilter{Pagination: types.Pagination{Page: 2, Size: 2}})
		require.NoError(t, err)
		require.Len(t, got, 1)

		count, err := plugins.Count(ctx, types.ListQueryFilter{})
		require.NoError(t, err)
		require.Equal(t, int64(3), count)

		count, err = plugins.Count(ctx, types.ListQueryFilter{Query: "docker"})
		require.NoError(t, err)
		require.Equal(t, int64(2), count)
	})

	t.Run("update", func(t *testing.T) {
		docker.Description = rstValUpdated
		docker.Spec = "spec: updated"
		require.NoError(t, plugins.Update(ctx, docker))

		got, err := plugins.Find(ctx, "docker", "1.0.0")
		require.NoError(t, err)
		require.Equal(t, rstValUpdated, got.Description)
		require.Equal(t, "spec: updated", got.Spec)
	})
}

func rstNewJob(uid string) *job.Job {
	return &job.Job{
		UID: uid, Created: 1, Updated: 1, Type: "test", Data: "data", Result: "", MaxDurationSeconds: 60,
		MaxRetries: 3, State: job.JobStateScheduled, Scheduled: 1000, GroupID: "group-a",
	}
}

func TestRstJobStore(t *testing.T) {
	db, teardown := setupDB(t)
	t.Cleanup(teardown)
	ctx := context.Background()
	jobs := database.NewJobStore(db)

	now := time.UnixMilli(10_000)

	t.Run("create find duplicate", func(t *testing.T) {
		j := rstNewJob("job-1")
		require.NoError(t, jobs.Create(ctx, j))

		got, err := jobs.Find(ctx, "job-1")
		require.NoError(t, err)
		require.Equal(t, "data", got.Data)
		require.Equal(t, job.JobStateScheduled, got.State)
		require.Equal(t, 3, got.MaxRetries)

		require.ErrorIs(t, jobs.Create(ctx, rstNewJob("job-1")), gitness_store.ErrDuplicate)

		_, err = jobs.Find(ctx, "missing")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("upsert only changes a modified definition", func(t *testing.T) {
		j := rstNewJob("job-upsert")
		require.NoError(t, jobs.Upsert(ctx, j))

		// same definition: the existing row, including its progress fields, is untouched.
		running := *j
		running.State = job.JobStateRunning
		running.Updated = 5
		require.NoError(t, jobs.Upsert(ctx, &running))
		got, err := jobs.Find(ctx, "job-upsert")
		require.NoError(t, err)
		require.Equal(t, job.JobStateScheduled, got.State)
		require.Equal(t, int64(1), got.Updated)

		changed := *j
		changed.Data = "other data"
		changed.Updated = 7
		changed.MaxRetries = 9
		require.NoError(t, jobs.Upsert(ctx, &changed))
		got, err = jobs.Find(ctx, "job-upsert")
		require.NoError(t, err)
		require.Equal(t, "other data", got.Data)
		require.Equal(t, 9, got.MaxRetries)
		require.Equal(t, int64(7), got.Updated)
	})

	t.Run("update definition and execution", func(t *testing.T) {
		j := rstNewJob("job-update")
		require.NoError(t, jobs.Create(ctx, j))

		j.Data = "new data"
		j.Priority = job.JobPriorityElevated
		j.IsRecurring = true
		j.RecurringCron = "* * * * *"
		j.GroupID = "group-b"
		j.Updated = 20
		require.NoError(t, jobs.UpdateDefinition(ctx, j))

		got, err := jobs.Find(ctx, "job-update")
		require.NoError(t, err)
		require.Equal(t, "new data", got.Data)
		require.Equal(t, job.JobPriorityElevated, got.Priority)
		require.True(t, got.IsRecurring)
		require.Equal(t, "group-b", got.GroupID)

		j.State = job.JobStateRunning
		j.RunBy = "worker"
		j.RunDeadline = 5_000
		j.TotalExecutions = 1
		j.LastExecuted = 3_000
		j.ConsecutiveFailures = 2
		j.LastFailureError = "bad"
		require.NoError(t, jobs.UpdateExecution(ctx, j))

		got, err = jobs.Find(ctx, "job-update")
		require.NoError(t, err)
		require.Equal(t, job.JobStateRunning, got.State)
		require.Equal(t, "worker", got.RunBy)
		require.Equal(t, 1, got.TotalExecutions)
		require.Equal(t, 2, got.ConsecutiveFailures)
		require.Equal(t, "bad", got.LastFailureError)

		missing := rstNewJob("nope")
		require.ErrorIs(t, jobs.UpdateDefinition(ctx, missing), gitness_store.ErrResourceNotFound)
		require.ErrorIs(t, jobs.UpdateExecution(ctx, missing), gitness_store.ErrResourceNotFound)
	})

	t.Run("update progress only for running jobs", func(t *testing.T) {
		j := rstNewJob("job-progress")
		require.NoError(t, jobs.Create(ctx, j))

		j.RunProgress = 40
		j.Result = "halfway"
		require.ErrorIs(t, jobs.UpdateProgress(ctx, j), gitness_store.ErrResourceNotFound)

		j.State = job.JobStateRunning
		require.NoError(t, jobs.UpdateExecution(ctx, j))
		require.NoError(t, jobs.UpdateProgress(ctx, j))

		got, err := jobs.Find(ctx, "job-progress")
		require.NoError(t, err)
		require.Equal(t, 40, got.RunProgress)
		require.Equal(t, "halfway", got.Result)
	})

	t.Run("scheduling queries", func(t *testing.T) {
		ready := rstNewJob("sched-ready")
		ready.Scheduled = 5_000
		later := rstNewJob("sched-later")
		later.Scheduled = 20_000
		latest := rstNewJob("sched-latest")
		latest.Scheduled = 30_000
		elevated := rstNewJob("sched-elevated")
		elevated.Scheduled = 6_000
		elevated.Priority = job.JobPriorityElevated
		overdue := rstNewJob("sched-overdue")
		overdue.State = job.JobStateRunning
		overdue.RunDeadline = 9_000
		for _, j := range []*job.Job{ready, later, latest, elevated, overdue} {
			j.GroupID = "scheduling"
			require.NoError(t, jobs.Create(ctx, j))
		}

		got, err := jobs.ListReady(ctx, now, 10)
		require.NoError(t, err)
		uids := []string{}
		for _, j := range got {
			if j.GroupID == "scheduling" {
				uids = append(uids, j.UID)
			}
		}
		// higher priority first, then by scheduled time.
		require.Equal(t, []string{"sched-elevated", "sched-ready"}, uids)

		limited, err := jobs.ListReady(ctx, now, 1)
		require.NoError(t, err)
		require.Len(t, limited, 1)

		exceeded, err := jobs.ListDeadlineExceeded(ctx, now)
		require.NoError(t, err)
		found := false
		for _, j := range exceeded {
			found = found || j.UID == "sched-overdue"
		}
		require.True(t, found)

		next, err := jobs.NextScheduledTime(ctx, now)
		require.NoError(t, err)
		require.Equal(t, time.UnixMilli(20_000), next)

		// jobs seeded by the migrations are scheduled around the current time, so look far into the future.
		next, err = jobs.NextScheduledTime(ctx, time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)
		require.True(t, next.IsZero(), next.String())

		running, err := jobs.CountRunning(ctx)
		require.NoError(t, err)
		require.GreaterOrEqual(t, running, 1)
	})

	t.Run("group queries", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			j := rstNewJob(fmt.Sprintf("grouped-%d", i))
			j.GroupID = "to-delete"
			require.NoError(t, jobs.Create(ctx, j))
		}

		got, err := jobs.ListByGroupID(ctx, "to-delete")
		require.NoError(t, err)
		require.Len(t, got, 3)

		deleted, err := jobs.DeleteByGroupID(ctx, "to-delete")
		require.NoError(t, err)
		require.Equal(t, int64(3), deleted)

		got, err = jobs.ListByGroupID(ctx, "to-delete")
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("delete old and by uid", func(t *testing.T) {
		mk := func(uid string, state job.State, lastExecuted int64, recurring bool) {
			j := rstNewJob(uid)
			j.GroupID = "cleanup"
			j.State = state
			j.LastExecuted = lastExecuted
			j.IsRecurring = recurring
			require.NoError(t, jobs.Create(ctx, j))
		}
		mk("old-finished", job.JobStateFinished, 100, false)
		mk("old-failed", job.JobStateFailed, 100, false)
		mk("old-canceled", job.JobStateCanceled, 100, false)
		mk("old-recurring", job.JobStateFinished, 100, true)
		mk("recent-finished", job.JobStateFinished, 9_000, false)
		mk("old-running", job.JobStateRunning, 100, false)

		deleted, err := jobs.DeleteOld(ctx, time.UnixMilli(1_000))
		require.NoError(t, err)
		require.Equal(t, int64(3), deleted)

		left, err := jobs.ListByGroupID(ctx, "cleanup")
		require.NoError(t, err)
		uids := []string{}
		for _, j := range left {
			uids = append(uids, j.UID)
		}
		require.ElementsMatch(t, []string{"old-recurring", "recent-finished", "old-running"}, uids)

		require.NoError(t, jobs.DeleteByUID(ctx, "recent-finished"))
		_, err = jobs.Find(ctx, "recent-finished")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})
}

func TestRstSecretStore(t *testing.T) {
	ctx, f := rstNewFixture(t)
	space := f.rstMkSpace(ctx, t, 0, "secrets")
	secrets := database.NewSecretStore(f.db)

	mk := func(ident string) *types.Secret {
		s := &types.Secret{
			Identifier: ident, Description: "desc " + ident, SpaceID: space.ID, CreatedBy: userID, Data: "payload",
			Created: 1, Updated: 1,
		}
		require.NoError(t, secrets.Create(ctx, s))
		require.NotZero(t, s.ID)

		return s
	}

	api := mk("api-token")
	mk("deploy-key")

	require.ErrorIs(t, secrets.Create(ctx, &types.Secret{
		Identifier: "api-token", SpaceID: space.ID, CreatedBy: userID,
	}), gitness_store.ErrDuplicate)

	got, err := secrets.Find(ctx, api.ID)
	require.NoError(t, err)
	require.Equal(t, "api-token", got.Identifier)
	require.Equal(t, "payload", got.Data)

	got, err = secrets.FindByIdentifier(ctx, space.ID, "deploy-key")
	require.NoError(t, err)
	require.Equal(t, "deploy-key", got.Identifier)

	_, err = secrets.Find(ctx, 9999)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	_, err = secrets.FindByIdentifier(ctx, space.ID, "missing")
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	stale := *api
	api.Description = rstValUpdated
	api.Data = "rotated"
	require.NoError(t, secrets.Update(ctx, api))
	require.Equal(t, int64(1), api.Version)

	got, err = secrets.Find(ctx, api.ID)
	require.NoError(t, err)
	require.Equal(t, "rotated", got.Data)
	require.Equal(t, rstValUpdated, got.Description)

	stale.Description = rstValStale
	require.ErrorIs(t, secrets.Update(ctx, &stale), gitness_store.ErrVersionConflict)

	calls := 0
	updated, err := secrets.UpdateOptLock(ctx, &stale, func(s *types.Secret) error {
		calls++
		s.Description = rstValOptLock
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Equal(t, "rotated", updated.Data)
	require.Equal(t, rstValOptLock, updated.Description)

	boom := errors.New("boom")
	_, err = secrets.UpdateOptLock(ctx, updated, func(*types.Secret) error { return boom })
	require.ErrorIs(t, err, boom)

	list, err := secrets.List(ctx, space.ID, types.ListQueryFilter{})
	require.NoError(t, err)
	require.Len(t, list, 2)

	list, err = secrets.List(ctx, space.ID, types.ListQueryFilter{Query: "DEPLOY"})
	require.NoError(t, err)
	require.Len(t, list, 1)

	list, err = secrets.ListAll(ctx, space.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)

	count, err := secrets.Count(ctx, space.ID, types.ListQueryFilter{})
	require.NoError(t, err)
	require.Equal(t, int64(2), count)

	count, err = secrets.Count(ctx, space.ID, types.ListQueryFilter{Query: "api"})
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	require.NoError(t, secrets.Delete(ctx, api.ID))
	_, err = secrets.Find(ctx, api.ID)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	require.NoError(t, secrets.DeleteByIdentifier(ctx, space.ID, "deploy-key"))
	count, err = secrets.Count(ctx, space.ID, types.ListQueryFilter{})
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestRstTemplateStore(t *testing.T) {
	ctx, f := rstNewFixture(t)
	space := f.rstMkSpace(ctx, t, 0, "templates")
	templates := database.NewTemplateStore(f.db)

	mk := func(ident string, typ enum.ResolverType) *types.Template {
		tmpl := &types.Template{
			Identifier: ident, Description: "desc " + ident, Type: typ, SpaceID: space.ID, Data: "data: " + ident,
			Created: 1, Updated: 1,
		}
		require.NoError(t, templates.Create(ctx, tmpl))
		require.NotZero(t, tmpl.ID)

		return tmpl
	}

	step := mk("build", enum.ResolverTypeStep)
	mk("build", enum.ResolverTypeStage)
	mk("lint", enum.ResolverTypeStep)

	require.ErrorIs(t, templates.Create(ctx, &types.Template{
		Identifier: "build", Type: enum.ResolverTypeStep, SpaceID: space.ID,
	}), gitness_store.ErrDuplicate)

	got, err := templates.Find(ctx, step.ID)
	require.NoError(t, err)
	require.Equal(t, "data: build", got.Data)

	got, err = templates.FindByIdentifierAndType(ctx, space.ID, "build", enum.ResolverTypeStage)
	require.NoError(t, err)
	require.Equal(t, enum.ResolverTypeStage, got.Type)

	_, err = templates.FindByIdentifierAndType(ctx, space.ID, "lint", enum.ResolverTypeStage)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	_, err = templates.Find(ctx, 9999)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	stale := *step
	step.Data = "data: changed"
	require.NoError(t, templates.Update(ctx, step))
	require.Equal(t, int64(1), step.Version)

	stale.Data = "data: stale"
	require.ErrorIs(t, templates.Update(ctx, &stale), gitness_store.ErrVersionConflict)

	calls := 0
	updated, err := templates.UpdateOptLock(ctx, &stale, func(tmpl *types.Template) error {
		calls++
		tmpl.Description = rstValOptLock
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Equal(t, "data: changed", updated.Data)

	boom := errors.New("boom")
	_, err = templates.UpdateOptLock(ctx, updated, func(*types.Template) error { return boom })
	require.ErrorIs(t, err, boom)

	list, err := templates.List(ctx, space.ID, types.ListQueryFilter{})
	require.NoError(t, err)
	require.Len(t, list, 3)

	list, err = templates.List(ctx, space.ID, types.ListQueryFilter{Query: "LIN"})
	require.NoError(t, err)
	require.Len(t, list, 1)

	count, err := templates.Count(ctx, space.ID, types.ListQueryFilter{})
	require.NoError(t, err)
	require.Equal(t, int64(3), count)

	count, err = templates.Count(ctx, space.ID, types.ListQueryFilter{Query: "build"})
	require.NoError(t, err)
	require.Equal(t, int64(2), count)

	require.NoError(t, templates.Delete(ctx, step.ID))
	_, err = templates.Find(ctx, step.ID)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	require.NoError(t, templates.DeleteByIdentifierAndType(ctx, space.ID, "build", enum.ResolverTypeStage))
	count, err = templates.Count(ctx, space.ID, types.ListQueryFilter{})
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestRstConnectorStore(t *testing.T) {
	ctx, f := rstNewFixture(t)
	space := f.rstMkSpace(ctx, t, 0, "connectors")
	secrets := database.NewSecretStore(f.db)
	connectors := database.NewConnectorStore(f.db, secrets)

	for _, ident := range []string{"token-a", "token-b"} {
		require.NoError(t, secrets.Create(ctx, &types.Secret{
			Identifier: ident, SpaceID: space.ID, CreatedBy: userID, Data: "payload", Created: 1, Updated: 1,
		}))
	}

	githubConnector := func(ident, secretIdent string) *types.Connector {
		return &types.Connector{
			Identifier: ident, Description: "desc " + ident, SpaceID: space.ID, CreatedBy: userID,
			Type: enum.ConnectorTypeGithub, LastTestStatus: enum.ConnectorStatusSuccess, Created: 1, Updated: 1,
			ConnectorConfig: types.ConnectorConfig{Github: &types.GithubConnectorData{
				APIURL: "https://api.github.com", Insecure: true,
				Auth: &types.ConnectorAuth{
					AuthType: enum.ConnectorAuthTypeBearer,
					Bearer:   &types.BearerTokenCreds{Token: types.SecretRef{Identifier: secretIdent}},
				},
			}},
		}
	}

	first := githubConnector("gh-one", "token-a")
	require.NoError(t, connectors.Create(ctx, first))
	require.NotZero(t, first.ID)
	require.NoError(t, connectors.Create(ctx, githubConnector("gh-two", "token-b")))

	t.Run("create rejects invalid input", func(t *testing.T) {
		missingSecret := githubConnector("gh-missing", "no-such-secret")
		require.ErrorIs(t, connectors.Create(ctx, missingSecret), gitness_store.ErrResourceNotFound)

		noConfig := &types.Connector{Identifier: "none", SpaceID: space.ID, Type: enum.ConnectorTypeGithub}
		require.Error(t, connectors.Create(ctx, noConfig))

		noAuth := githubConnector("gh-noauth", "token-a")
		noAuth.Github.Auth = nil
		require.Error(t, connectors.Create(ctx, noAuth))

		basicAuth := githubConnector("gh-basic", "token-a")
		basicAuth.Github.Auth.AuthType = enum.ConnectorAuthTypeBasic
		require.Error(t, connectors.Create(ctx, basicAuth))

		require.ErrorIs(t, connectors.Create(ctx, githubConnector("gh-one", "token-a")), gitness_store.ErrDuplicate)
	})

	t.Run("find resolves the secret reference", func(t *testing.T) {
		got, err := connectors.Find(ctx, first.ID)
		require.NoError(t, err)
		require.Equal(t, "gh-one", got.Identifier)
		require.Equal(t, enum.ConnectorTypeGithub, got.Type)
		require.Equal(t, enum.ConnectorStatusSuccess, got.LastTestStatus)
		require.Equal(t, "https://api.github.com", got.Github.APIURL)
		require.True(t, got.Github.Insecure)
		require.Equal(t, enum.ConnectorAuthTypeBearer, got.Github.Auth.AuthType)
		require.Equal(t, "token-a", got.Github.Auth.Bearer.Token.Identifier)

		got, err = connectors.FindByIdentifier(ctx, space.ID, "gh-two")
		require.NoError(t, err)
		require.Equal(t, "token-b", got.Github.Auth.Bearer.Token.Identifier)

		_, err = connectors.Find(ctx, 9999)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
		_, err = connectors.FindByIdentifier(ctx, space.ID, "missing")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("update and opt lock", func(t *testing.T) {
		stale := *first

		first.Description = rstValUpdated
		first.LastTestStatus = enum.ConnectorStatusFailed
		first.LastTestErrorMsg = "connection refused"
		first.Github.Auth.Bearer.Token.Identifier = "token-b"
		require.NoError(t, connectors.Update(ctx, first))
		require.Equal(t, int64(1), first.Version)

		got, err := connectors.Find(ctx, first.ID)
		require.NoError(t, err)
		require.Equal(t, rstValUpdated, got.Description)
		require.Equal(t, enum.ConnectorStatusFailed, got.LastTestStatus)
		require.Equal(t, "connection refused", got.LastTestErrorMsg)
		require.Equal(t, "token-b", got.Github.Auth.Bearer.Token.Identifier)

		stale.Description = rstValStale
		require.ErrorIs(t, connectors.Update(ctx, &stale), gitness_store.ErrVersionConflict)

		calls := 0
		updated, err := connectors.UpdateOptLock(ctx, &stale, func(c *types.Connector) error {
			calls++
			c.Description = rstValOptLock
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 2, calls)
		require.Equal(t, rstValOptLock, updated.Description)

		boom := errors.New("boom")
		_, err = connectors.UpdateOptLock(ctx, updated, func(*types.Connector) error { return boom })
		require.ErrorIs(t, err, boom)

		missingSecret := *updated
		missingSecret.Github = &types.GithubConnectorData{
			Auth: &types.ConnectorAuth{
				AuthType: enum.ConnectorAuthTypeBearer,
				Bearer:   &types.BearerTokenCreds{Token: types.SecretRef{Identifier: "gone"}},
			},
		}
		require.ErrorIs(t, connectors.Update(ctx, &missingSecret), gitness_store.ErrResourceNotFound)
	})

	t.Run("list and count", func(t *testing.T) {
		got, err := connectors.List(ctx, space.ID, types.ListQueryFilter{})
		require.NoError(t, err)
		require.Len(t, got, 2)

		got, err = connectors.List(ctx, space.ID, types.ListQueryFilter{Query: "TWO"})
		require.NoError(t, err)
		require.Len(t, got, 1)

		got, err = connectors.List(ctx, space.ID, types.ListQueryFilter{Pagination: types.Pagination{Page: 2, Size: 1}})
		require.NoError(t, err)
		require.Len(t, got, 1)

		count, err := connectors.Count(ctx, space.ID, types.ListQueryFilter{})
		require.NoError(t, err)
		require.Equal(t, int64(2), count)

		count, err = connectors.Count(ctx, space.ID, types.ListQueryFilter{Query: "gh-o"})
		require.NoError(t, err)
		require.Equal(t, int64(1), count)
	})

	t.Run("stored rows with other auth types and invalid data", func(t *testing.T) {
		basicSecret, err := secrets.FindByIdentifier(ctx, space.ID, "token-a")
		require.NoError(t, err)

		insert := func(ident, connType, authType string, username, password, token any) int64 {
			var id int64
			err := f.db.QueryRowContext(ctx, `
				INSERT INTO connectors (
					connector_description, connector_type, connector_created_by, connector_space_id,
					connector_identifier, connector_last_test_attempt, connector_last_test_error_msg,
					connector_last_test_status, connector_created, connector_updated, connector_version,
					connector_auth_type, connector_username, connector_password, connector_token
				) VALUES ('', ?, ?, ?, ?, 0, '', 'success', 1, 1, 0, ?, ?, ?, ?) RETURNING connector_id`,
				connType, userID, space.ID, ident, authType, username, password, token).Scan(&id)
			require.NoError(t, err)

			return id
		}

		basicID := insert("raw-basic", "github", "basic", "alice", basicSecret.ID, nil)
		got, err := connectors.Find(ctx, basicID)
		require.NoError(t, err)
		require.Equal(t, enum.ConnectorAuthTypeBasic, got.Github.Auth.AuthType)
		require.Equal(t, "alice", got.Github.Auth.Basic.Username)
		require.Equal(t, "token-a", got.Github.Auth.Basic.Password.Identifier)

		invalid := map[string]int64{
			"basic without password":     insert("raw-basic-nopw", "github", "basic", "alice", nil, nil),
			"bearer without token":       insert("raw-bearer-notoken", "github", "bearer", nil, nil, nil),
			"unknown auth type":          insert("raw-unknown-auth", "github", "unknown", nil, nil, nil),
			"unsupported connector type": insert("raw-unknown-type", "gitlab", "bearer", nil, nil, basicSecret.ID),
		}
		for name, id := range invalid {
			_, err := connectors.Find(ctx, id)
			require.Error(t, err, name)
		}
	})

	t.Run("delete", func(t *testing.T) {
		require.NoError(t, connectors.Delete(ctx, first.ID))
		_, err := connectors.Find(ctx, first.ID)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

		require.NoError(t, connectors.DeleteByIdentifier(ctx, space.ID, "gh-two"))
		_, err = connectors.FindByIdentifier(ctx, space.ID, "gh-two")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})
}

func TestRstAITaskStore(t *testing.T) {
	ctx, f := rstNewCDEFixture(t)
	infra := f.rstMkInfraConfig(ctx, t, "docker-cfg", enum.InfraProviderTypeDocker)
	resource := f.rstMkInfraResource(ctx, t, infra.ID, "small", 100)
	cfg := f.rstMkGitspace(ctx, t, "cfg", resource.ID)
	instance := f.rstMkInstance(ctx, t, cfg.ID, enum.GitspaceInstanceStateRunning, 100)
	cfg.GitspaceInstance = instance

	tasks := database.NewAITaskStore(f.db)

	mk := func(ident string, state enum.AITaskState, created int64) *types.AITask {
		task := &types.AITask{
			Identifier: ident, GitspaceConfig: cfg, InitialPrompt: "fix " + ident, DisplayName: "Task " + ident,
			UserUID: "user_1", SpaceID: f.spaceID, Created: created, Updated: created,
			AIAgent: enum.AIAgentClaudeCode, State: state,
		}
		require.NoError(t, tasks.Create(ctx, task))
		require.NotZero(t, task.ID)

		return task
	}

	first := mk("alpha", enum.AITaskStateRunning, 100)
	mk("beta", enum.AITaskStateCompleted, 200)
	mk("gamma", enum.AITaskStateError, 300)

	t.Run("duplicate identifier", func(t *testing.T) {
		err := tasks.Create(ctx, &types.AITask{
			Identifier: "alpha", GitspaceConfig: cfg, SpaceID: f.spaceID, AIAgent: enum.AIAgentClaudeCode,
			State: enum.AITaskStateRunning,
		})
		require.ErrorIs(t, err, gitness_store.ErrDuplicate)
	})

	t.Run("find", func(t *testing.T) {
		got, err := tasks.Find(ctx, first.ID)
		require.NoError(t, err)
		require.Equal(t, "alpha", got.Identifier)
		require.Equal(t, cfg.ID, got.GitspaceConfigID)
		require.Equal(t, instance.ID, got.GitspaceInstanceID)
		require.Equal(t, "fix alpha", got.InitialPrompt)
		require.Nil(t, got.APIURL)

		got, err = tasks.FindByIdentifier(ctx, f.spaceID, "ALPHA")
		require.NoError(t, err)
		require.Equal(t, first.ID, got.ID)

		_, err = tasks.Find(ctx, 9999)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
		_, err = tasks.FindByIdentifier(ctx, f.spaceID, "missing")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("update", func(t *testing.T) {
		url := "https://agent.example.com"
		output := "done"
		msg := "warning"
		first.DisplayName = "Renamed"
		first.APIURL = &url
		first.State = enum.AITaskStateCompleted
		first.Output = &output
		first.ErrorMessage = &msg
		first.OutputMetadata = json.RawMessage(`{"tokens":12}`)
		first.Updated = 500
		require.NoError(t, tasks.Update(ctx, first))

		got, err := tasks.Find(ctx, first.ID)
		require.NoError(t, err)
		require.Equal(t, "Renamed", got.DisplayName)
		require.Equal(t, url, *got.APIURL)
		require.Equal(t, enum.AITaskStateCompleted, got.State)
		require.Equal(t, output, *got.Output)
		require.Equal(t, msg, *got.ErrorMessage)
		require.JSONEq(t, `{"tokens":12}`, string(got.OutputMetadata))
		require.Equal(t, int64(500), got.Updated)
	})

	t.Run("list and count", func(t *testing.T) {
		base := types.AITaskFilter{SpaceID: f.spaceID, UserIdentifier: "user_1"}

		got, err := tasks.List(ctx, &base)
		require.NoError(t, err)
		// newest first
		require.Equal(t, []string{"gamma", "beta", "alpha"},
			[]string{got[0].Identifier, got[1].Identifier, got[2].Identifier})

		tests := []struct {
			name   string
			mutate func(*types.AITaskFilter)
			want   int
		}{
			{"all", func(*types.AITaskFilter) {}, 3},
			{"state", func(f *types.AITaskFilter) { f.States = []enum.AITaskState{enum.AITaskStateError} }, 1},
			{"agent", func(f *types.AITaskFilter) { f.AIAgents = []enum.AIAgent{enum.AIAgentClaudeCode} }, 3},
			{"query by prompt", func(f *types.AITaskFilter) { f.QueryFilter.Query = "FIX GAM" }, 1},
			{"query by name", func(f *types.AITaskFilter) { f.QueryFilter.Query = rstValRenamed }, 1},
			{"other user", func(f *types.AITaskFilter) { f.UserIdentifier = "someone" }, 0},
			{"other space", func(f *types.AITaskFilter) { f.SpaceID = 9999 }, 0},
			{"paging", func(f *types.AITaskFilter) { f.QueryFilter.Pagination = types.Pagination{Page: 2, Size: 2} }, 1},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				filter := base
				tt.mutate(&filter)

				listed, err := tasks.List(ctx, &filter)
				require.NoError(t, err)
				require.Len(t, listed, tt.want)

				if filter.QueryFilter.Size == 0 {
					count, err := tasks.Count(ctx, &filter)
					require.NoError(t, err)
					require.Equal(t, int64(tt.want), count)
				}
			})
		}
	})
}
