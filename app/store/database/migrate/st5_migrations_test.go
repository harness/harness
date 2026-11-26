// Copyright 2026 Harness, Inc.
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

package migrate

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/harness/gitness/app/services/protection"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3" // sqlite driver
)

const (
	st5LastVersion  = "0193"
	st5FirstVersion = "0001_create_table_a_principals"
	st5SqliteDriver = "sqlite3"
	st5ErrBoom      = "st5boom"
	st5DockerType   = "DOCKER"
	st5RPMType      = "RPM"
	st5DigestDigits = "1234"
)

// st5RawDB opens an empty file based SQLite database with foreign keys disabled.
func st5RawDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Connect(st5SqliteDriver, filepath.Join(t.TempDir(), "st5.db"))
	require.NoError(t, err)
	// a single connection keeps the pragma in effect for every statement.
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	return db
}

// st5MigratedDB returns a fully migrated database with foreign keys disabled.
func st5MigratedDB(ctx context.Context, t *testing.T) *sqlx.DB {
	t.Helper()

	db := st5RawDB(t)
	require.NoError(t, Migrate(ctx, db))
	_, err := db.ExecContext(ctx, `PRAGMA foreign_keys = OFF`)
	require.NoError(t, err)

	return db
}

func st5Exec(ctx context.Context, t *testing.T, db *sqlx.DB, query string, args ...any) {
	t.Helper()

	_, err := db.ExecContext(ctx, query, args...)
	require.NoError(t, err, query)
}

// st5InTx runs fn in a transaction and commits it when fn succeeds.
func st5InTx(ctx context.Context, db *sqlx.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err = fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

func st5FailTrigger(ctx context.Context, t *testing.T, db *sqlx.DB, table, event string) {
	t.Helper()

	st5Exec(ctx, t, db, `CREATE TRIGGER st5_fail_`+table+`_`+event+` BEFORE `+event+` ON `+table+
		` BEGIN SELECT RAISE(ABORT, '`+st5ErrBoom+`'); END`)
}

func st5InsertWebhook(ctx context.Context, t *testing.T, db *sqlx.DB, repoID, spaceID any, name string, uid any) {
	t.Helper()

	st5Exec(ctx, t, db, `INSERT INTO webhooks (webhook_created_by, webhook_created, webhook_updated, webhook_space_id,
		webhook_repo_id, webhook_display_name, webhook_description, webhook_url, webhook_secret, webhook_enabled,
		webhook_insecure, webhook_triggers, webhook_uid) VALUES (1, 1, 1, ?, ?, ?, '', 'http://x', '', 1, 0, '[]', ?)`,
		spaceID, repoID, name, uid)
}

func st5WebhookUIDs(ctx context.Context, t *testing.T, db *sqlx.DB) []string {
	t.Helper()

	var uids []string
	require.NoError(t, db.SelectContext(ctx, &uids, `SELECT webhook_uid FROM webhooks ORDER BY webhook_id`))

	return uids
}

func TestSt5Migrate0039WebhookIdentifiers(t *testing.T) {
	ctx := context.Background()
	db := st5MigratedDB(ctx, t)

	st5InsertWebhook(ctx, t, db, 1, nil, "My Hook", nil)
	st5InsertWebhook(ctx, t, db, 1, nil, "My Hook", nil)
	st5InsertWebhook(ctx, t, db, 1, nil, "Has UID", "existing")
	st5InsertWebhook(ctx, t, db, 1, nil, " .-_ ", nil)
	st5InsertWebhook(ctx, t, db, 2, nil, "My Hook", nil)
	st5InsertWebhook(ctx, t, db, nil, 5, "My Hook", nil)

	require.NoError(t, st5InTx(ctx, db, func(tx *sql.Tx) error {
		return migrateAfter_0039_alter_table_webhooks_uid(ctx, tx)
	}))

	uids := st5WebhookUIDs(ctx, t, db)
	require.Len(t, uids, 6)
	require.Equal(t, "My_Hook", uids[0])
	require.Regexp(t, regexp.MustCompile(`^My_Hook_[a-z0-9]{4}$`), uids[1])
	require.Equal(t, "existing", uids[2])
	require.Regexp(t, regexp.MustCompile(`^webhook_[a-z0-9]{4}$`), uids[3])
	require.Equal(t, "My_Hook", uids[4])
	require.Equal(t, "My_Hook", uids[5])

	// running the migration again must not change anything.
	require.NoError(t, st5InTx(ctx, db, func(tx *sql.Tx) error {
		return migrateAfter_0039_alter_table_webhooks_uid(ctx, tx)
	}))
	require.Equal(t, uids, st5WebhookUIDs(ctx, t, db))
}

func TestSt5Migrate0039Errors(t *testing.T) {
	ctx := context.Background()

	t.Run("update fails", func(t *testing.T) {
		db := st5MigratedDB(ctx, t)
		st5InsertWebhook(ctx, t, db, 1, nil, "Hook", nil)
		st5FailTrigger(ctx, t, db, "webhooks", "UPDATE")

		err := st5InTx(ctx, db, func(tx *sql.Tx) error {
			return migrateAfter_0039_alter_table_webhooks_uid(ctx, tx)
		})
		require.ErrorContains(t, err, st5ErrBoom)
	})

	t.Run("select fails", func(t *testing.T) {
		db := st5MigratedDB(ctx, t)
		st5Exec(ctx, t, db, `DROP TABLE webhooks`)

		err := st5InTx(ctx, db, func(tx *sql.Tx) error {
			return migrateAfter_0039_alter_table_webhooks_uid(ctx, tx)
		})
		require.ErrorContains(t, err, "failed to read next batch of webhooks")
	})

	t.Run("cancelled context", func(t *testing.T) {
		db := st5MigratedDB(ctx, t)
		cancelled, cancel := context.WithCancel(ctx)
		cancel()

		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)
		defer func() { _ = tx.Rollback() }()

		require.Error(t, migrateAfter_0039_alter_table_webhooks_uid(cancelled, tx))
	})
}

func st5InsertRule(ctx context.Context, t *testing.T, db *sqlx.DB, uid, ruleType, definition string) {
	t.Helper()

	st5Exec(ctx, t, db, `INSERT INTO rules (rule_version, rule_created_by, rule_created, rule_updated, rule_repo_id,
		rule_uid, rule_description, rule_type, rule_state, rule_pattern, rule_definition)
		VALUES (0, 1, 1, 1, 1, ?, '', ?, 'active', '{}', ?)`, uid, ruleType, definition)
}

func st5RuleDefinition(ctx context.Context, t *testing.T, db *sqlx.DB, uid string) string {
	t.Helper()

	var def string
	require.NoError(t, db.GetContext(ctx, &def, `SELECT rule_definition FROM rules WHERE rule_uid = ?`, uid))

	return def
}

func TestSt5Migrate0042BranchRules(t *testing.T) {
	ctx := context.Background()
	db := st5MigratedDB(ctx, t)

	unchanged, err := protection.ToJSON(&protection.Branch{})
	require.NoError(t, err)

	legacy := `{"pullreq":{"status_checks":{"require_uids":["ci"]}}}`
	st5InsertRule(ctx, t, db, "legacy", "branch", legacy)
	st5InsertRule(ctx, t, db, "current", "branch", string(unchanged))
	// non-branch rules must not be touched, even if their definition is not valid for a branch.
	st5InsertRule(ctx, t, db, "other", "tag", `not json`)

	require.NoError(t, st5InTx(ctx, db, func(tx *sql.Tx) error {
		return migrateAfter_0042_alter_table_rules(ctx, tx)
	}))

	migrated := protection.Branch{}
	require.NoError(t, json.Unmarshal([]byte(st5RuleDefinition(ctx, t, db, "legacy")), &migrated))
	require.Equal(t, []string{"ci"}, migrated.PullReq.StatusChecks.RequireIdentifiers)
	require.Contains(t, st5RuleDefinition(ctx, t, db, "legacy"), "require_identifiers")
	require.Equal(t, string(unchanged), st5RuleDefinition(ctx, t, db, "current"))
	require.Equal(t, "not json", st5RuleDefinition(ctx, t, db, "other"))
}

func TestSt5Migrate0042Errors(t *testing.T) {
	ctx := context.Background()
	run := func(db *sqlx.DB) error {
		return st5InTx(ctx, db, func(tx *sql.Tx) error { return migrateAfter_0042_alter_table_rules(ctx, tx) })
	}

	t.Run("invalid definition", func(t *testing.T) {
		db := st5MigratedDB(ctx, t)
		st5InsertRule(ctx, t, db, "bad", "branch", `not json`)
		require.ErrorContains(t, run(db), "failed to unmarshal branch definition")
	})

	t.Run("update fails", func(t *testing.T) {
		db := st5MigratedDB(ctx, t)
		st5InsertRule(ctx, t, db, "legacy", "branch", `{"pullreq":{"status_checks":{"require_uids":["ci"]}}}`)
		st5FailTrigger(ctx, t, db, "rules", "UPDATE")
		require.ErrorContains(t, run(db), st5ErrBoom)
	})

	t.Run("select fails", func(t *testing.T) {
		db := st5MigratedDB(ctx, t)
		st5Exec(ctx, t, db, `DROP TABLE rules`)
		require.ErrorContains(t, run(db), "failed to read next batch of rules")
	})
}

func st5InsertRegistry(ctx context.Context, t *testing.T, db *sqlx.DB, name, pkgType string) int64 {
	t.Helper()

	var id int64
	require.NoError(t, db.GetContext(ctx, &id, `INSERT INTO registries (registry_name, registry_root_parent_id,
		registry_parent_id, registry_type, registry_package_type, registry_created_at, registry_updated_at,
		registry_created_by, registry_updated_by, registry_uuid) VALUES (?, 1, 1, 'VIRTUAL', ?, 1, 1, 1, 1, ?)
		RETURNING registry_id`, name, pkgType, "uuid-"+name))

	return id
}

func st5InsertImage(ctx context.Context, t *testing.T, db *sqlx.DB, regID int64, name string) int64 {
	t.Helper()

	var id int64
	require.NoError(t, db.GetContext(ctx, &id, `INSERT INTO images (image_name, image_registry_id, image_created_at,
		image_updated_at, image_created_by, image_updated_by, image_uuid) VALUES (?, ?, 1, 1, 1, 1, ?)
		RETURNING image_id`, name, regID, "uuid-img-"+strconv.FormatInt(regID, 10)+name))

	return id
}

func st5InsertManifest(ctx context.Context, t *testing.T, db *sqlx.DB, regID int64, image string, digest []byte) {
	t.Helper()

	st5Exec(ctx, t, db, `INSERT INTO manifests (manifest_registry_id, manifest_schema_version, manifest_media_type_id,
		manifest_total_size, manifest_digest, manifest_payload, manifest_image_name, manifest_created_at,
		manifest_created_by, manifest_updated_at, manifest_updated_by) VALUES (?, 2, 1, 1, ?, x'00', ?, 7, 8, 9, 10)`,
		regID, digest, image)
}

func st5ArtifactVersions(ctx context.Context, t *testing.T, db *sqlx.DB) []string {
	t.Helper()

	var versions []string
	require.NoError(t, db.SelectContext(ctx, &versions, `SELECT artifact_version FROM artifacts ORDER BY artifact_id`))

	return versions
}

func TestSt5Migrate0153Artifacts(t *testing.T) {
	ctx := context.Background()
	db := st5MigratedDB(ctx, t)

	dockerID := st5InsertRegistry(ctx, t, db, "docker", st5DockerType)
	rpmID := st5InsertRegistry(ctx, t, db, "rpm", st5RPMType)
	st5InsertImage(ctx, t, db, dockerID, "app")
	st5InsertImage(ctx, t, db, rpmID, "app")
	digest := []byte{0x12, 0x34}
	st5InsertManifest(ctx, t, db, dockerID, "app", digest)
	// manifests of registries other than docker/helm are ignored.
	st5InsertManifest(ctx, t, db, rpmID, "app", digest)

	require.NoError(t, st5InTx(ctx, db, func(tx *sql.Tx) error {
		return MigrateAfter_0153_migrate_artifacts(ctx, tx, st5SqliteDriver)
	}))

	require.Equal(t, []string{st5DigestDigits}, st5ArtifactVersions(ctx, t, db))

	// second run finds no orphans.
	require.NoError(t, st5InTx(ctx, db, func(tx *sql.Tx) error {
		return MigrateAfter_0153_migrate_artifacts(ctx, tx, st5SqliteDriver)
	}))
	require.Len(t, st5ArtifactVersions(ctx, t, db), 1)
}

func TestSt5ProcessManifestsBatch(t *testing.T) {
	ctx := context.Background()
	db := st5MigratedDB(ctx, t)

	regID := st5InsertRegistry(ctx, t, db, "docker", st5DockerType)
	st5InsertImage(ctx, t, db, regID, "known")
	manifests := []manifest{
		{ID: 5, ImageName: "known", RegistryID: regID, Digest: []byte{0xab, 0xcd}, CreatedAt: 1, CreatedBy: 2, UpdatedBy: 3},
		// the image does not exist yet and gets created.
		{ID: 9, ImageName: "fresh", RegistryID: regID, Digest: []byte{0x01}, CreatedAt: 1, CreatedBy: 2, UpdatedBy: 3},
	}

	var last int64
	run := func() error {
		return st5InTx(ctx, db, func(tx *sql.Tx) (err error) {
			last, err = processManifestsBatch(ctx, tx, manifests)
			return err
		})
	}
	require.NoError(t, run())
	require.EqualValues(t, 9, last)
	require.Equal(t, []string{"abcd", "01"}, st5ArtifactVersions(ctx, t, db))

	var imageCount int
	require.NoError(t, db.GetContext(ctx, &imageCount, `SELECT count(*) FROM images`))
	require.Equal(t, 2, imageCount)

	// existing artifacts are left alone.
	require.NoError(t, run())
	require.Len(t, st5ArtifactVersions(ctx, t, db), 2)

	got, err := st5GetManifestsByIDs(ctx, db, nil)
	require.NoError(t, err)
	require.Nil(t, got)
}

func st5GetManifestsByIDs(ctx context.Context, db *sqlx.DB, ids []int64) (res []manifest, err error) {
	err = st5InTx(ctx, db, func(tx *sql.Tx) error {
		res, err = getManifestsByIDs(ctx, tx, ids)
		return err
	})

	return res, err
}

func TestSt5Migrate0153Errors(t *testing.T) {
	ctx := context.Background()
	manifests := []manifest{{ID: 1, ImageName: "app", RegistryID: 1, Digest: []byte{0x01}}}
	process := func(db *sqlx.DB) error {
		return st5InTx(ctx, db, func(tx *sql.Tx) error {
			_, err := processManifestsBatch(ctx, tx, manifests)
			return err
		})
	}

	t.Run("orphan query fails", func(t *testing.T) {
		db := st5MigratedDB(ctx, t)
		st5Exec(ctx, t, db, `DROP TABLE artifacts`)
		err := st5InTx(ctx, db, func(tx *sql.Tx) error {
			return MigrateAfter_0153_migrate_artifacts(ctx, tx, st5SqliteDriver)
		})
		require.ErrorContains(t, err, "failed to get orphan manifest IDs")
	})

	t.Run("postgres flavoured orphan query fails on sqlite", func(t *testing.T) {
		db := st5MigratedDB(ctx, t)
		err := st5InTx(ctx, db, func(tx *sql.Tx) error {
			return MigrateAfter_0153_migrate_artifacts(ctx, tx, "postgres")
		})
		require.ErrorContains(t, err, "failed to get orphan manifest IDs")
	})

	t.Run("manifests query fails", func(t *testing.T) {
		db := st5MigratedDB(ctx, t)
		st5Exec(ctx, t, db, `DROP TABLE manifests`)
		_, err := st5GetManifestsByIDs(ctx, db, []int64{1})
		require.ErrorContains(t, err, "failed to query manifests by IDs")
	})

	t.Run("image lookup fails", func(t *testing.T) {
		db := st5MigratedDB(ctx, t)
		st5Exec(ctx, t, db, `DROP TABLE images`)
		require.ErrorContains(t, process(db), "failed to check for existing image")
	})

	t.Run("image insert fails", func(t *testing.T) {
		db := st5MigratedDB(ctx, t)
		st5FailTrigger(ctx, t, db, "images", "INSERT")
		require.ErrorContains(t, process(db), "failed to insert image")
	})

	t.Run("artifact lookup fails", func(t *testing.T) {
		db := st5MigratedDB(ctx, t)
		st5InsertImage(ctx, t, db, 1, "app")
		st5Exec(ctx, t, db, `DROP TABLE artifacts`)
		require.ErrorContains(t, process(db), "failed to check for existing artifact")
	})

	t.Run("artifact insert fails", func(t *testing.T) {
		db := st5MigratedDB(ctx, t)
		st5InsertImage(ctx, t, db, 1, "app")
		st5FailTrigger(ctx, t, db, "artifacts", "INSERT")
		require.ErrorContains(t, process(db), "failed to insert artifact")
	})
}

func st5RPMMetadata(t *testing.T, epoch string) string {
	t.Helper()

	meta := map[string]any{
		"file_metadata": map[string]any{"epoch": epoch},
		"files":         []map[string]any{{"file_name": "old.rpm"}},
	}
	raw, err := json.Marshal(meta)
	require.NoError(t, err)

	return string(raw)
}

func st5InsertArtifact(
	ctx context.Context, t *testing.T, db *sqlx.DB, imageID int64, version, metadata string,
) int64 {
	t.Helper()

	var id int64
	require.NoError(t, db.GetContext(ctx, &id, `INSERT INTO artifacts (artifact_version, artifact_image_id,
		artifact_created_at, artifact_updated_at, artifact_created_by, artifact_updated_by, artifact_metadata,
		artifact_uuid) VALUES (?, ?, 1, 1, 1, 1, ?, ?) RETURNING artifact_id`,
		version, imageID, metadata, "uuid-art-"+version))

	return id
}

func st5InsertNode(
	ctx context.Context, t *testing.T, db *sqlx.DB, id, name string, parent any, regID int64, isFile bool, path string,
) {
	t.Helper()

	st5Exec(ctx, t, db, `INSERT INTO nodes (node_id, node_name, node_parent_id, node_registry_id, node_is_file,
		node_path, node_created_at, node_created_by) VALUES (?, ?, ?, ?, ?, ?, 11, 12)`,
		id, name, parent, regID, isFile, path)
}

func st5RPMFixture(ctx context.Context, t *testing.T, db *sqlx.DB) (artifactID, regID int64) {
	t.Helper()

	regID = st5InsertRegistry(ctx, t, db, "rpm", st5RPMType)
	imageID := st5InsertImage(ctx, t, db, regID, "foo")
	artifactID = st5InsertArtifact(ctx, t, db, imageID, "1-1.0.5.x86_64", st5RPMMetadata(t, "1"))
	st5InsertNode(ctx, t, db, "dir", "1-1.0.5", nil, regID, false, "/foo/1-1.0.5")
	st5InsertNode(ctx, t, db, "file", "foo-1-1.0.5.x86_64.rpm", "dir", regID, true,
		"/foo/1-1.0.5/foo-1-1.0.5.x86_64.rpm")

	return artifactID, regID
}

func st5ProcessArtifacts(ctx context.Context, db *sqlx.DB, artifacts []artifact) error {
	return st5InTx(ctx, db, func(tx *sql.Tx) error { return processArtifactsBatch(ctx, tx, artifacts) })
}

func st5Artifact(id, regID int64, version, epoch string, t *testing.T) artifact {
	t.Helper()

	return artifact{ID: id, Name: "foo", Version: version, RegistryID: regID,
		Metadata: json.RawMessage(st5RPMMetadata(t, epoch))}
}

func TestSt5Migrate0155ProcessArtifactsBatch(t *testing.T) {
	ctx := context.Background()
	db := st5MigratedDB(ctx, t)

	artifactID, regID := st5RPMFixture(ctx, t, db)
	noPrefixImage := st5InsertImage(ctx, t, db, regID, "baz")
	noPrefixID := st5InsertArtifact(ctx, t, db, noPrefixImage, "2.0.0.noarch", st5RPMMetadata(t, "7"))

	require.NoError(t, st5ProcessArtifacts(ctx, db, []artifact{
		st5Artifact(artifactID, regID, "1-1.0.5.x86_64", "1", t),
		// the version prefix does not match the epoch: only the index job is scheduled.
		st5Artifact(noPrefixID, regID, "2.0.0.noarch", "7", t),
	}))

	var versions []string
	require.NoError(t, db.SelectContext(ctx, &versions,
		`SELECT artifact_version FROM artifacts ORDER BY artifact_id`))
	require.Equal(t, []string{"1:1.0.5.x86_64", "2.0.0.noarch"}, versions)

	var meta string
	require.NoError(t, db.GetContext(ctx, &meta,
		`SELECT artifact_metadata FROM artifacts WHERE artifact_id = ?`, artifactID))
	require.Contains(t, meta, "foo-1.0.5.x86_64.rpm")

	type nodeRow struct {
		ID     string  `db:"node_id"`
		Name   string  `db:"node_name"`
		Path   string  `db:"node_path"`
		Parent *string `db:"node_parent_id"`
	}
	var nodes []nodeRow
	require.NoError(t, db.SelectContext(ctx, &nodes,
		`SELECT node_id, node_name, node_path, node_parent_id FROM nodes ORDER BY node_path`))
	require.Len(t, nodes, 3)

	byID := map[string]nodeRow{}
	var epochNode nodeRow
	for _, n := range nodes {
		byID[n.ID] = n
		if n.Name == "1" {
			epochNode = n
		}
	}
	require.Equal(t, "/foo/1.0.5", byID["dir"].Path)
	require.Equal(t, "1.0.5", byID["dir"].Name)
	require.Equal(t, "/foo/1.0.5/1", epochNode.Path)
	require.Equal(t, "/foo/1.0.5/1/foo-1.0.5.x86_64.rpm", byID["file"].Path)
	require.Equal(t, "foo-1.0.5.x86_64.rpm", byID["file"].Name)
	require.NotNil(t, byID["file"].Parent)
	require.Equal(t, epochNode.ID, *byID["file"].Parent)

	// an index job is scheduled for every processed artifact.
	var jobs []string
	require.NoError(t, db.SelectContext(ctx, &jobs,
		`SELECT job_data FROM jobs WHERE job_type = 'rpm_registry_index' AND job_data LIKE '%[%'`))
	require.Len(t, jobs, 2)
	for _, data := range jobs {
		in := RegistrySyncInput{}
		require.NoError(t, json.Unmarshal([]byte(data), &in))
		require.Equal(t, []int64{regID}, in.RegistryIDs)
	}
}

func TestSt5Migrate0155ProcessErrors(t *testing.T) {
	ctx := context.Background()
	fixture := func(t *testing.T) (*sqlx.DB, []artifact) {
		t.Helper()

		db := st5MigratedDB(ctx, t)
		id, regID := st5RPMFixture(ctx, t, db)

		return db, []artifact{st5Artifact(id, regID, "1-1.0.5.x86_64", "1", t)}
	}

	t.Run("invalid metadata", func(t *testing.T) {
		db, arts := fixture(t)
		arts[0].Metadata = json.RawMessage(`{"files":"x"}`)
		require.Error(t, st5ProcessArtifacts(ctx, db, arts))
	})

	t.Run("metadata update fails", func(t *testing.T) {
		db, arts := fixture(t)
		st5FailTrigger(ctx, t, db, "artifacts", "UPDATE")
		require.ErrorContains(t, st5ProcessArtifacts(ctx, db, arts), "failed to update artifact metadata")
	})

	t.Run("nodes query fails", func(t *testing.T) {
		db, arts := fixture(t)
		st5Exec(ctx, t, db, `DROP TABLE nodes`)
		require.ErrorContains(t, st5ProcessArtifacts(ctx, db, arts), "failed to query nodes")
	})

	t.Run("node update fails", func(t *testing.T) {
		db, arts := fixture(t)
		st5FailTrigger(ctx, t, db, "nodes", "UPDATE")
		require.ErrorContains(t, st5ProcessArtifacts(ctx, db, arts), "failed to update node")
	})

	t.Run("epoch node insert fails", func(t *testing.T) {
		db, arts := fixture(t)
		st5FailTrigger(ctx, t, db, "nodes", "INSERT")
		require.ErrorContains(t, st5ProcessArtifacts(ctx, db, arts), "failed to insert node")
	})

	t.Run("job insert fails", func(t *testing.T) {
		db, arts := fixture(t)
		st5Exec(ctx, t, db, `DROP TABLE jobs`)
		require.ErrorContains(t, st5ProcessArtifacts(ctx, db, arts), "failed to schedule rpm registry index job")
	})
}

func TestSt5Migrate0155ArtifactsBatch(t *testing.T) {
	ctx := context.Background()
	run := func(db *sqlx.DB) error {
		return st5InTx(ctx, db, func(tx *sql.Tx) error {
			return MigrateAfter_0155_migrate_rpm_artifacts(ctx, tx, st5SqliteDriver)
		})
	}

	t.Run("nothing to migrate", func(t *testing.T) {
		db := st5MigratedDB(ctx, t)
		regID := st5InsertRegistry(ctx, t, db, "rpm", st5RPMType)
		imageID := st5InsertImage(ctx, t, db, regID, "bar")
		st5InsertArtifact(ctx, t, db, imageID, "9.9.9", st5RPMMetadata(t, "0"))
		require.NoError(t, run(db))
	})

	t.Run("artifacts query fails", func(t *testing.T) {
		db := st5MigratedDB(ctx, t)
		st5Exec(ctx, t, db, `DROP TABLE artifacts`)
		require.ErrorContains(t, run(db), "failed to get artifacts batch")
	})

	t.Run("sqlite text metadata cannot be scanned", func(t *testing.T) {
		// the query is written for Postgres where jsonb is returned as bytes; SQLite returns text.
		db := st5MigratedDB(ctx, t)
		st5RPMFixture(ctx, t, db)
		require.ErrorContains(t, run(db), "failed to scan artifact")
	})
}

func TestSt5Migrate0160(t *testing.T) {
	ctx := context.Background()
	db := st5MigratedDB(ctx, t)

	rpmA := st5InsertRegistry(ctx, t, db, "rpm-a", st5RPMType)
	rpmB := st5InsertRegistry(ctx, t, db, "rpm-b", st5RPMType)
	st5InsertRegistry(ctx, t, db, "docker", st5DockerType)

	// the migration run by Migrate itself scheduled a job without registries.
	st5Exec(ctx, t, db, `DELETE FROM jobs`)

	require.NoError(t, st5InTx(ctx, db, func(tx *sql.Tx) error { return MigrateAfter_0160(ctx, tx) }))

	var data []string
	require.NoError(t, db.SelectContext(ctx, &data, `SELECT job_data FROM jobs WHERE job_type = 'rpm_registry_index'`))
	require.Len(t, data, 1)
	in := RegistrySyncInput{}
	require.NoError(t, json.Unmarshal([]byte(data[0]), &in))
	require.ElementsMatch(t, []int64{rpmA, rpmB}, in.RegistryIDs)
}

func TestSt5Migrate0160Errors(t *testing.T) {
	ctx := context.Background()

	t.Run("registries query fails", func(t *testing.T) {
		db := st5MigratedDB(ctx, t)
		st5Exec(ctx, t, db, `DROP TABLE registries`)
		err := st5InTx(ctx, db, func(tx *sql.Tx) error { return MigrateAfter_0160(ctx, tx) })
		require.ErrorContains(t, err, "failed to query registries")
	})

	t.Run("job insert fails", func(t *testing.T) {
		db := st5MigratedDB(ctx, t)
		st5Exec(ctx, t, db, `DROP TABLE jobs`)
		err := st5InTx(ctx, db, func(tx *sql.Tx) error { return MigrateAfter_0160(ctx, tx) })
		require.ErrorContains(t, err, "failed to schedule index job")
	})
}

func TestSt5MigrateCurrentAndTo(t *testing.T) {
	ctx := context.Background()
	db := st5RawDB(t)

	version, err := Current(ctx, db)
	require.NoError(t, err)
	require.Empty(t, version)

	require.NoError(t, To(ctx, db, st5FirstVersion))
	version, err = Current(ctx, db)
	require.NoError(t, err)
	require.Equal(t, st5FirstVersion, version)

	require.NoError(t, Migrate(ctx, db))
	version, err = Current(ctx, db)
	require.NoError(t, err)
	require.Equal(t, st5LastVersion, version)

	// foreign keys must be re-enabled after a migration.
	var fk int
	require.NoError(t, db.GetContext(ctx, &fk, `PRAGMA foreign_keys`))
	require.Equal(t, 1, fk)
}

func TestSt5MigrateCurrentErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("unsupported driver", func(t *testing.T) {
		db := sqlx.NewDb(st5RawDB(t).DB, "mysql")
		_, err := Current(ctx, db)
		require.ErrorContains(t, err, "unsupported driver")
		require.ErrorContains(t, Migrate(ctx, db), "failed to get migrator")
		require.ErrorContains(t, To(ctx, db, "x"), "failed to get migrator")
	})

	t.Run("postgres query on sqlite", func(t *testing.T) {
		db := sqlx.NewDb(st5RawDB(t).DB, "postgres")
		_, err := Current(ctx, db)
		require.ErrorContains(t, err, "failed to check migration table existence")
	})

	t.Run("empty migrations table", func(t *testing.T) {
		db := st5RawDB(t)
		st5Exec(ctx, t, db, `CREATE TABLE migrations (version TEXT)`)
		_, err := Current(ctx, db)
		require.ErrorContains(t, err, "failed to read current DB version")
	})

	t.Run("closed database", func(t *testing.T) {
		db := st5RawDB(t)
		require.NoError(t, db.Close())
		_, err := Current(ctx, db)
		require.Error(t, err)
		require.Error(t, Migrate(ctx, db))
		require.Error(t, To(ctx, db, st5FirstVersion))
	})
}
