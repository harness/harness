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
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

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
	connector_identifier,
	connector_description,
	connector_type,
	connector_auth_type,
	connector_created_by,
	connector_space_id,
	connector_last_test_attempt,
	connector_last_test_error_msg,
	connector_last_test_status,
	connector_created,
	connector_updated,
	connector_version,
	connector_address,
	connector_insecure,
	connector_username,
	connector_github_app_installation_id,
	connector_github_app_application_id,
	connector_region,
	connector_password,
	connector_token,
	connector_aws_key,
	connector_aws_secret,
	connector_github_app_private_key,
	connector_token_refresh
	`
)

type connector struct {
	ID               int64  `db:"connector_id"`
	Identifier       string `db:"connector_identifier"`
	Description      string `db:"connector_description"`
	Type             string `db:"connector_type"`
	AuthType         string `db:"connector_auth_type"`
	CreatedBy        int64  `db:"connector_created_by"`
	SpaceID          int64  `db:"connector_space_id"`
	LastTestAttempt  int64  `db:"connector_last_test_attempt"`
	LastTestErrorMsg string `db:"connector_last_test_error_msg"`
	LastTestStatus   string `db:"connector_last_test_status"`
	Created          int64  `db:"connector_created"`
	Updated          int64  `db:"connector_updated"`
	Version          int64  `db:"connector_version"`

	Address                 sql.NullString `db:"connector_address"`
	Insecure                sql.NullBool   `db:"connector_insecure"`
	Username                sql.NullString `db:"connector_username"`
	GithubAppInstallationID sql.NullString `db:"connector_github_app_installation_id"`
	GithubAppApplicationID  sql.NullString `db:"connector_github_app_application_id"`
	Region                  sql.NullString `db:"connector_region"`
	// Password fields are stored as reference to secrets table
	Password            sql.NullInt64 `db:"connector_password"`
	Token               sql.NullInt64 `db:"connector_token"`
	AWSKey              sql.NullInt64 `db:"connector_aws_key"`
	AWSSecret           sql.NullInt64 `db:"connector_aws_secret"`
	GithubAppPrivateKey sql.NullInt64 `db:"connector_github_app_private_key"`
	TokenRefresh        sql.NullInt64 `db:"connector_token_refresh"`
}

// NewConnectorStore returns a new ConnectorStore.
// The secret store is used to resolve the secret references.
func NewConnectorStore(db *sqlx.DB, secretStore store.SecretStore) store.ConnectorStore {
	return &connectorStore{
		db:          db,
		secretStore: secretStore,
	}
}

func (s *connectorStore) mapFromDBConnectors(ctx context.Context, src []*connector) ([]*types.Connector, error) {
	dst := make([]*types.Connector, len(src))
	for i, v := range src {
		m, err := s.mapFromDBConnector(ctx, v)
		if err != nil {
			return nil, fmt.Errorf("could not map from db connector: %w", err)
		}
		dst[i] = m
	}
	return dst, nil
}

func (s *connectorStore) mapToDBConnector(ctx context.Context, v *types.Connector) (*connector, error) {
	to := connector{
		ID:               v.ID,
		Identifier:       v.Identifier,
		Description:      v.Description,
		Type:             v.Type.String(),
		SpaceID:          v.SpaceID,
		CreatedBy:        v.CreatedBy,
		Created:          v.Created,
		Updated:          v.Updated,
		Version:          v.Version,
		LastTestAttempt:  v.LastTestAttempt,
		LastTestErrorMsg: v.LastTestErrorMsg,
		LastTestStatus:   v.LastTestStatus.String(),
	}
	// Parse connector specific configs
	err := s.convertConfigToDB(ctx, v, &to)
	if err != nil {
		return nil, fmt.Errorf("could not convert config to db: %w", err)
	}
	return &to, nil
}

func (s *connectorStore) convertConfigToDB(
	ctx context.Context,
	source *types.Connector,
	to *connector,
) error {
	switch {
	case source.Github != nil:
		to.Address = sql.NullString{String: source.Github.APIURL, Valid: true}
		to.Insecure = sql.NullBool{Bool: source.Github.Insecure, Valid: true}
		if source.Github.Auth == nil {
			return fmt.Errorf("auth is required for github connectors")
		}
		if source.Github.Auth.AuthType != enum.ConnectorAuthTypeBearer {
			return fmt.Errorf("only bearer token auth is supported for github connectors")
		}
		to.AuthType = source.Github.Auth.AuthType.String()
		creds := source.Github.Auth.Bearer
		// use the same space ID as the connector
		tokenID, err := s.secretIdentiferToID(ctx, creds.Token.Identifier, source.SpaceID)
		if err != nil {
			return fmt.Errorf("could not find secret: %w", err)
		}
		to.Token = sql.NullInt64{Int64: tokenID, Valid: true}
	default:
		return fmt.Errorf("no connector config found for type: %s", source.Type)
	}
	return nil
}

// secretIdentiferToID finds the secret ID given the space ID and the identifier.
func (s *connectorStore) secretIdentiferToID(
	ctx context.Context,
	identifier string,
	spaceID int64,
) (int64, error) {
	secret, err := s.secretStore.FindByIdentifier(ctx, spaceID, identifier)
	if err != nil {
		return 0, err
	}
	return secret.ID, nil
}

func (s *connectorStore) mapFromDBConnector(
	ctx context.Context,
	dbConnector *connector,
) (*types.Connector, error) {
	connector := &types.Connector{
		ID:               dbConnector.ID,
		Identifier:       dbConnector.Identifier,
		Description:      dbConnector.Description,
		Type:             enum.ConnectorType(dbConnector.Type),
		SpaceID:          dbConnector.SpaceID,
		CreatedBy:        dbConnector.CreatedBy,
		LastTestAttempt:  dbConnector.LastTestAttempt,
		LastTestErrorMsg: dbConnector.LastTestErrorMsg,
		LastTestStatus:   enum.ConnectorStatus(dbConnector.LastTestStatus),
		Created:          dbConnector.Created,
		Updated:          dbConnector.Updated,
		Version:          dbConnector.Version,
	}
	err := s.populateConnectorData(ctx, dbConnector, connector)
	if err != nil {
		return nil, fmt.Errorf("could not populate connector data: %w", err)
	}
	return connector, nil
}

func (s *connectorStore) populateConnectorData(
	ctx context.Context,
	source *connector,
	to *types.Connector,
) error {
	switch enum.ConnectorType(source.Type) {
	case enum.ConnectorTypeGithub:
		githubData, err := s.parseGithubConnectorData(ctx, source)
		if err != nil {
			return fmt.Errorf("could not parse github connector data: %w", err)
		}
		to.Github = githubData
	// Cases for other connectors can be added here
	default:
		return fmt.Errorf("unsupported connector type: %s", source.Type)
	}
	return nil
}

func (s *connectorStore) parseGithubConnectorData(
	ctx context.Context,
	connector *connector,
) (*types.GithubConnectorData, error) {
	auth, err := s.parseAuthenticationData(ctx, connector)
	if err != nil {
		return nil, fmt.Errorf("could not parse authentication data: %w", err)
	}
	return &types.GithubConnectorData{
		APIURL:   connector.Address.String,
		Insecure: connector.Insecure.Bool,
		Auth:     auth,
	}, nil
}

func (s *connectorStore) parseAuthenticationData(
	ctx context.Context,
	connector *connector,
) (*types.ConnectorAuth, error) {
	authType, err := enum.ParseConnectorAuthType(connector.AuthType)
	if err != nil {
		return nil, err
	}

	switch authType {
	case enum.ConnectorAuthTypeBasic:
		if !connector.Username.Valid || !connector.Password.Valid {
			return nil, fmt.Errorf("basic auth requires both username and password")
		}
		passwordRef, err := s.convertToRef(ctx, connector.Password.Int64)
		if err != nil {
			return nil, fmt.Errorf("could not convert basicauth password to ref: %w", err)
		}
		return &types.ConnectorAuth{
			AuthType: enum.ConnectorAuthTypeBasic,
			Basic: &types.BasicAuthCreds{
				Username: connector.Username.String,
				Password: passwordRef,
			},
		}, nil
	case enum.ConnectorAuthTypeBearer:
		if !connector.Token.Valid {
			return nil, fmt.Errorf("bearer auth requires a token")
		}
		tokenRef, err := s.convertToRef(ctx, connector.Token.Int64)
		if err != nil {
			return nil, fmt.Errorf("could not convert bearer token to ref: %w", err)
		}
		return &types.ConnectorAuth{
			AuthType: enum.ConnectorAuthTypeBearer,
			Bearer: &types.BearerTokenCreds{
				Token: tokenRef,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported auth type: %s", connector.AuthType)
	}
}

func (s *connectorStore) convertToRef(ctx context.Context, id int64) (types.SecretRef, error) {
	secret, err := s.secretStore.Find(ctx, id)
	if err != nil {
		return types.SecretRef{}, err
	}
	return types.SecretRef{
		Identifier: secret.Identifier,
	}, nil
}

type connectorStore struct {
	db          *sqlx.DB
	secretStore store.SecretStore
}

// Find returns a connector given a connector ID.
func (s *connectorStore) Find(ctx context.Context, id int64) (*types.Connector, error) {
	const findQueryStmt = connectorQueryBase + `
		WHERE connector_id = $1`
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(connector)
	if err := db.GetContext(ctx, dst, findQueryStmt, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find connector")
	}
	return s.mapFromDBConnector(ctx, dst)
}

// FindByIdentifier returns a connector in a given space with a given identifier.
func (s *connectorStore) FindByIdentifier(
	ctx context.Context,
	spaceID int64,
	identifier string,
) (*types.Connector, error) {
	const findQueryStmt = connectorQueryBase + `
		WHERE connector_space_id = $1 AND connector_identifier = $2`
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(connector)
	if err := db.GetContext(ctx, dst, findQueryStmt, spaceID, identifier); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find connector")
	}
	return s.mapFromDBConnector(ctx, dst)
}

// Create creates a connector.
func (s *connectorStore) Create(ctx context.Context, connector *types.Connector) error {
	dbConnector, err := s.mapToDBConnector(ctx, connector)
	if err != nil {
		return err
	}
	const connectorInsertStmt = `
	INSERT INTO connectors (
		connector_description
		,connector_type
		,connector_created_by
		,connector_space_id
		,connector_identifier
		,connector_last_test_attempt
		,connector_last_test_error_msg
		,connector_last_test_status
		,connector_created
		,connector_updated
		,connector_version
		,connector_auth_type
		,connector_address
		,connector_insecure
		,connector_username
		,connector_github_app_installation_id
		,connector_github_app_application_id
		,connector_region
		,connector_password
		,connector_token
		,connector_aws_key
		,connector_aws_secret
		,connector_github_app_private_key
		,connector_token_refresh
	) VALUES (
		:connector_description
		,:connector_type
		,:connector_created_by
		,:connector_space_id
		,:connector_identifier
		,:connector_last_test_attempt
		,:connector_last_test_error_msg
		,:connector_last_test_status
		,:connector_created
		,:connector_updated
		,:connector_version
		,:connector_auth_type
		,:connector_address
		,:connector_insecure
		,:connector_username
		,:connector_github_app_installation_id
		,:connector_github_app_application_id
		,:connector_region
		,:connector_password
		,:connector_token
		,:connector_aws_key
		,:connector_aws_secret
		,:connector_github_app_private_key
		,:connector_token_refresh
	) RETURNING connector_id`
	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(connectorInsertStmt, dbConnector)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind connector object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&connector.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "connector query failed")
	}

	return nil
}

func (s *connectorStore) Update(ctx context.Context, p *types.Connector) error {
	conn, err := s.mapToDBConnector(ctx, p)
	if err != nil {
		return err
	}
	const connectorUpdateStmt = `
	UPDATE connectors
	SET
		connector_description = :connector_description
		,connector_identifier = :connector_identifier
		,connector_last_test_attempt = :connector_last_test_attempt
		,connector_last_test_error_msg = :connector_last_test_error_msg
		,connector_last_test_status = :connector_last_test_status
		,connector_updated = :connector_updated
		,connector_version = :connector_version
		,connector_auth_type = :connector_auth_type
		,connector_address = :connector_address
		,connector_insecure = :connector_insecure
		,connector_username = :connector_username
		,connector_github_app_installation_id = :connector_github_app_installation_id
		,connector_github_app_application_id = :connector_github_app_application_id
		,connector_region = :connector_region
		,connector_password = :connector_password
		,connector_token = :connector_token
		,connector_aws_key = :connector_aws_key
		,connector_aws_secret = :connector_aws_secret
		,connector_github_app_private_key = :connector_github_app_private_key
		,connector_token_refresh = :connector_token_refresh
	WHERE connector_id = :connector_id AND connector_version = :connector_version - 1`
	o := *conn

	o.Version++
	o.Updated = time.Now().UnixMilli()

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(connectorUpdateStmt, o)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind connector object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update connector")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	p.Version = o.Version
	p.Updated = o.Updated
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
		stmt = stmt.Where("LOWER(connector_identifier) LIKE ?", fmt.Sprintf("%%%s%%", strings.ToLower(filter.Query)))
	}

	stmt = stmt.Limit(database.Limit(filter.Size))
	stmt = stmt.Offset(database.Offset(filter.Page, filter.Size))

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*connector{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}

	return s.mapFromDBConnectors(ctx, dst)
}

// Delete deletes a connector given a connector ID.
func (s *connectorStore) Delete(ctx context.Context, id int64) error {
	const connectorDeleteStmt = `
		DELETE FROM connectors
		WHERE connector_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, connectorDeleteStmt, id); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Could not delete connector")
	}

	return nil
}

// DeleteByIdentifier deletes a connector with a given identifier in a space.
func (s *connectorStore) DeleteByIdentifier(ctx context.Context, spaceID int64, identifier string) error {
	const connectorDeleteStmt = `
	DELETE FROM connectors
	WHERE connector_space_id = $1 AND connector_identifier = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, connectorDeleteStmt, spaceID, identifier); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Could not delete connector")
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
		stmt = stmt.Where("LOWER(connector_identifier) LIKE ?", fmt.Sprintf("%%%s%%", filter.Query))
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}
	return count, nil
}
