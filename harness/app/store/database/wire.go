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

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/database/migrate"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/store/database"

	"github.com/google/wire"
	"github.com/jmoiron/sqlx"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideDatabase,
	ProvidePrincipalStore,
	ProvideUserGroupStore,
	ProvideUserGroupReviewerStore,
	ProvidePrincipalInfoView,
	ProvideInfraProviderResourceView,
	ProvideSpacePathStore,
	ProvideSpaceStore,
	ProvideRepoStore,
	ProvideBranchStore,
	ProvideRuleStore,
	ProvideJobStore,
	ProvideExecutionStore,
	ProvidePipelineStore,
	ProvideStageStore,
	ProvideStepStore,
	ProvideSecretStore,
	ProvideMembershipStore,
	ProvideTokenStore,
	ProvidePullReqStore,
	ProvidePullReqActivityStore,
	ProvideCodeCommentView,
	ProvidePullReqReviewStore,
	ProvidePullReqReviewerStore,
	ProvidePullReqFileViewStore,
	ProvideWebhookStore,
	ProvideWebhookExecutionStore,
	ProvideSettingsStore,
	ProvidePublicAccessStore,
	ProvideCheckStore,
	ProvideConnectorStore,
	ProvideTemplateStore,
	ProvideTriggerStore,
	ProvidePluginStore,
	ProvidePublicKeyStore,
	ProvidePublicKeySubKeyStore,
	ProvideGitSignatureResultStore,
	ProvideInfraProviderConfigStore,
	ProvideInfraProviderResourceStore,
	ProvideGitspaceConfigStore,
	ProvideGitspaceInstanceStore,
	ProvideGitspaceEventStore,
	ProvideLabelStore,
	ProvideLabelValueStore,
	ProvidePullReqLabelStore,
	ProvideLFSObjectStore,
	ProvideInfraProviderTemplateStore,
	ProvideInfraProvisionedStore,
	ProvideUsageMetricStore,
	ProvideCDEGatewayStore,
	ProvideFavoriteStore,
	ProvideGitspaceSettingsStore,
)

// migrator is helper function to set up the database by performing automated
// database migration steps.
func migrator(ctx context.Context, db *sqlx.DB) error {
	return migrate.Migrate(ctx, db)
}

// ProvideDatabase provides a database connection.
func ProvideDatabase(ctx context.Context, config database.Config) (*sqlx.DB, error) {
	return database.ConnectAndMigrate(
		ctx,
		config.Driver,
		config.Datasource,
		migrator,
	)
}

// ProvidePrincipalStore provides a principal store.
func ProvidePrincipalStore(db *sqlx.DB, uidTransformation store.PrincipalUIDTransformation) store.PrincipalStore {
	return NewPrincipalStore(db, uidTransformation)
}

// ProvideUserGroupStore provides a principal store.
func ProvideUserGroupStore(db *sqlx.DB) store.UserGroupStore {
	return NewUserGroupStore(db)
}

// ProvideUserGroupReviewerStore provides a usergroup reviewer store.
func ProvideUserGroupReviewerStore(
	db *sqlx.DB,
	pInfoCache store.PrincipalInfoCache,
	userGroupStore store.UserGroupStore,
) store.UserGroupReviewerStore {
	return NewUsergroupReviewerStore(db, pInfoCache, userGroupStore)
}

// ProvidePrincipalInfoView provides a principal info store.
func ProvidePrincipalInfoView(db *sqlx.DB) store.PrincipalInfoView {
	return NewPrincipalInfoView(db)
}

// ProvideInfraProviderResourceView provides a principal info store.
func ProvideInfraProviderResourceView(
	db *sqlx.DB, spaceStore store.SpaceStore,
) store.InfraProviderResourceView {
	return NewInfraProviderResourceView(db, spaceStore)
}

// ProvideSpacePathStore provides a space path store.
func ProvideSpacePathStore(
	db *sqlx.DB,
	spacePathTransformation store.SpacePathTransformation,
) store.SpacePathStore {
	return NewSpacePathStore(db, spacePathTransformation)
}

// ProvideSpaceStore provides a space store.
func ProvideSpaceStore(
	db *sqlx.DB,
	spacePathCache store.SpacePathCache,
	spacePathStore store.SpacePathStore,
) store.SpaceStore {
	return NewSpaceStore(db, spacePathCache, spacePathStore)
}

// ProvideRepoStore provides a repo store.
func ProvideRepoStore(
	db *sqlx.DB,
	spacePathCache store.SpacePathCache,
	spacePathStore store.SpacePathStore,
	spaceStore store.SpaceStore,
) store.RepoStore {
	return NewRepoStore(db, spacePathCache, spacePathStore, spaceStore)
}

// ProvideRuleStore provides a rule store.
func ProvideRuleStore(
	db *sqlx.DB,
	principalInfoCache store.PrincipalInfoCache,
) store.RuleStore {
	return NewRuleStore(db, principalInfoCache)
}

// ProvideJobStore provides a job store.
func ProvideJobStore(db *sqlx.DB) job.Store {
	return NewJobStore(db)
}

// ProvidePipelineStore provides a pipeline store.
func ProvidePipelineStore(db *sqlx.DB) store.PipelineStore {
	return NewPipelineStore(db)
}

// ProvideInfraProviderConfigStore provides a infraprovider config store.
func ProvideInfraProviderConfigStore(
	db *sqlx.DB,
	spaceIDCache store.SpaceIDCache,
) store.InfraProviderConfigStore {
	return NewInfraProviderConfigStore(db, spaceIDCache)
}

// ProvideGitspaceInstanceStore provides a infraprovider resource store.
func ProvideInfraProviderResourceStore(
	db *sqlx.DB,
	spaceIDCache store.SpaceIDCache,
) store.InfraProviderResourceStore {
	return NewInfraProviderResourceStore(db, spaceIDCache)
}

// ProvideGitspaceConfigStore provides a gitspace config store.
func ProvideGitspaceConfigStore(
	db *sqlx.DB,
	pCache store.PrincipalInfoCache,
	rCache store.InfraProviderResourceCache,
	spaceIDCache store.SpaceIDCache,
) store.GitspaceConfigStore {
	return NewGitspaceConfigStore(db, pCache, rCache, spaceIDCache)
}

// ProvideGitspaceSettingsStore provides a gitspace settings store.
func ProvideGitspaceSettingsStore(db *sqlx.DB) store.GitspaceSettingsStore {
	return NewGitspaceSettingsStore(db)
}

// ProvideGitspaceInstanceStore provides a gitspace instance store.
func ProvideGitspaceInstanceStore(
	db *sqlx.DB,
	spaceIDCache store.SpaceIDCache,
) store.GitspaceInstanceStore {
	return NewGitspaceInstanceStore(db, spaceIDCache)
}

// ProvideStageStore provides a stage store.
func ProvideStageStore(db *sqlx.DB) store.StageStore {
	return NewStageStore(db)
}

// ProvideStepStore provides a step store.
func ProvideStepStore(db *sqlx.DB) store.StepStore {
	return NewStepStore(db)
}

// ProvideSecretStore provides a secret store.
func ProvideSecretStore(db *sqlx.DB) store.SecretStore {
	return NewSecretStore(db)
}

// ProvideConnectorStore provides a connector store.
func ProvideConnectorStore(db *sqlx.DB, secretStore store.SecretStore) store.ConnectorStore {
	return NewConnectorStore(db, secretStore)
}

// ProvideTemplateStore provides a template store.
func ProvideTemplateStore(db *sqlx.DB) store.TemplateStore {
	return NewTemplateStore(db)
}

// ProvideTriggerStore provides a trigger store.
func ProvideTriggerStore(db *sqlx.DB) store.TriggerStore {
	return NewTriggerStore(db)
}

// ProvideExecutionStore provides an execution store.
func ProvideExecutionStore(db *sqlx.DB) store.ExecutionStore {
	return NewExecutionStore(db)
}

// ProvidePluginStore provides a plugin store.
func ProvidePluginStore(db *sqlx.DB) store.PluginStore {
	return NewPluginStore(db)
}

func ProvideMembershipStore(
	db *sqlx.DB,
	principalInfoCache store.PrincipalInfoCache,
	spacePathStore store.SpacePathStore,
	spaceStore store.SpaceStore,
) store.MembershipStore {
	return NewMembershipStore(db, principalInfoCache, spacePathStore, spaceStore)
}

// ProvideTokenStore provides a token store.
func ProvideTokenStore(db *sqlx.DB) store.TokenStore {
	return NewTokenStore(db)
}

// ProvidePullReqStore provides a pull request store.
func ProvidePullReqStore(
	db *sqlx.DB,
	principalInfoCache store.PrincipalInfoCache,
) store.PullReqStore {
	return NewPullReqStore(db, principalInfoCache)
}

// ProvidePullReqActivityStore provides a pull request activity store.
func ProvidePullReqActivityStore(
	db *sqlx.DB,
	principalInfoCache store.PrincipalInfoCache,
) store.PullReqActivityStore {
	return NewPullReqActivityStore(db, principalInfoCache)
}

// ProvideCodeCommentView provides a code comment view.
func ProvideCodeCommentView(db *sqlx.DB) store.CodeCommentView {
	return NewCodeCommentView(db)
}

// ProvidePullReqReviewStore provides a pull request review store.
func ProvidePullReqReviewStore(db *sqlx.DB) store.PullReqReviewStore {
	return NewPullReqReviewStore(db)
}

// ProvidePullReqReviewerStore provides a pull request reviewer store.
func ProvidePullReqReviewerStore(
	db *sqlx.DB,
	principalInfoCache store.PrincipalInfoCache,
) store.PullReqReviewerStore {
	return NewPullReqReviewerStore(db, principalInfoCache)
}

// ProvidePullReqFileViewStore provides a pull request file view store.
func ProvidePullReqFileViewStore(db *sqlx.DB) store.PullReqFileViewStore {
	return NewPullReqFileViewStore(db)
}

// ProvideWebhookStore provides a webhook store.
func ProvideWebhookStore(db *sqlx.DB) store.WebhookStore {
	return NewWebhookStore(db)
}

// ProvideWebhookExecutionStore provides a webhook execution store.
func ProvideWebhookExecutionStore(db *sqlx.DB) store.WebhookExecutionStore {
	return NewWebhookExecutionStore(db)
}

// ProvideCheckStore provides a status check result store.
func ProvideCheckStore(
	db *sqlx.DB,
	principalInfoCache store.PrincipalInfoCache,
) store.CheckStore {
	return NewCheckStore(db, principalInfoCache)
}

// ProvideSettingsStore provides a settings store.
func ProvideSettingsStore(db *sqlx.DB) store.SettingsStore {
	return NewSettingsStore(db)
}

// ProvidePublicAccessStore provides a public access store.
func ProvidePublicAccessStore(db *sqlx.DB) store.PublicAccessStore {
	return NewPublicAccessStore(db)
}

// ProvidePublicKeyStore provides a public key store.
func ProvidePublicKeyStore(db *sqlx.DB) store.PublicKeyStore {
	return NewPublicKeyStore(db)
}

// ProvidePublicKeySubKeyStore provides a public key sub key store.
func ProvidePublicKeySubKeyStore(db *sqlx.DB) store.PublicKeySubKeyStore {
	return NewPublicKeySubKeyStore(db)
}

func ProvideGitSignatureResultStore(db *sqlx.DB) store.GitSignatureResultStore {
	return NewGitSignatureResultStore(db)
}

// ProvideBranchStore provides a branch store.
func ProvideBranchStore(db *sqlx.DB) store.BranchStore {
	return NewBranchStore(db)
}

// ProvideGitspaceEventStore provides a gitspace event store.
func ProvideGitspaceEventStore(db *sqlx.DB) store.GitspaceEventStore {
	return NewGitspaceEventStore(db)
}

// ProvideLabelStore provides a label store.
func ProvideLabelStore(db *sqlx.DB) store.LabelStore {
	return NewLabelStore(db)
}

// ProvideLabelValueStore provides a label value store.
func ProvideLabelValueStore(db *sqlx.DB) store.LabelValueStore {
	return NewLabelValueStore(db)
}

// ProvideLabelValueStore provides a label value store.
func ProvidePullReqLabelStore(db *sqlx.DB) store.PullReqLabelAssignmentStore {
	return NewPullReqLabelStore(db)
}

// ProvideLFSObjectStore provides an lfs object store.
func ProvideLFSObjectStore(db *sqlx.DB) store.LFSObjectStore {
	return NewLFSObjectStore(db)
}

// ProvideInfraProviderTemplateStore provides a infraprovider template store.
func ProvideInfraProviderTemplateStore(db *sqlx.DB) store.InfraProviderTemplateStore {
	return NewInfraProviderTemplateStore(db)
}

// ProvideInfraProvisionedStore provides a provisioned infra store.
func ProvideInfraProvisionedStore(db *sqlx.DB) store.InfraProvisionedStore {
	return NewInfraProvisionedStore(db)
}

func ProvideUsageMetricStore(db *sqlx.DB) store.UsageMetricStore {
	return NewUsageMetricsStore(db)
}

func ProvideCDEGatewayStore(db *sqlx.DB) store.CDEGatewayStore {
	return NewCDEGatewayStore(db)
}

func ProvideFavoriteStore(db *sqlx.DB) store.FavoriteStore {
	return NewFavoriteStore(db)
}
