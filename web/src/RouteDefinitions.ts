/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { CDERoutes, routes as cdeRoutes } from 'cde/RouteDefinitions'

export interface CODEProps {
  space?: string
  repoName?: string
  repoPath?: string
  gitRef?: string
  resourcePath?: string
  commitRef?: string
  branch?: string
  tags?: string
  diffRefs?: string
  pullRequestId?: string
  pullRequestSection?: string
  webhookId?: string
  pipeline?: string
  execution?: string
  commitSHA?: string
  secret?: string
  settingSection?: string
  ruleId?: string
  settingSectionMode?: string
  gitspaceId?: string
}

export interface CODEQueryProps {
  query?: string
}

export const pathProps: Readonly<Omit<Required<CODEProps>, 'repoPath' | 'branch' | 'tags'>> = {
  space: ':space*',
  repoName: ':repoName',
  gitRef: ':gitRef*',
  resourcePath: ':resourcePath*',
  commitRef: ':commitRef*',
  diffRefs: ':diffRefs*',
  pullRequestId: ':pullRequestId',
  pullRequestSection: ':pullRequestSection',
  webhookId: ':webhookId',
  pipeline: ':pipeline',
  execution: ':execution',
  commitSHA: ':commitSHA',
  secret: ':secret',
  settingSection: ':settingSection',
  ruleId: ':ruleId',
  settingSectionMode: ':settingSectionMode',
  gitspaceId: ':gitspaceId'
}

export interface CODERoutes extends CDERoutes {
  toSignIn: () => string
  toRegister: () => string

  toCODEHome: () => string

  toCODESpaceAccessControl: (args: Required<Pick<CODEProps, 'space'>>) => string
  toCODESpaceSettings: (args: Required<Pick<CODEProps, 'space'>>) => string
  toCODEPipelines: (args: Required<Pick<CODEProps, 'repoPath'>>) => string
  toCODEPipelineEdit: (args: Required<Pick<CODEProps, 'repoPath' | 'pipeline'>>) => string
  toCODEPipelineSettings: (args: Required<Pick<CODEProps, 'repoPath' | 'pipeline'>>) => string
  toCODESecrets: (args: Required<Pick<CODEProps, 'space'>>) => string

  toCODEGlobalSettings: () => string
  toCODEUsers: () => string
  toCODEUserProfile: () => string
  toCODEUserChangePassword: () => string

  toCODERepositories: (args: Required<Pick<CODEProps, 'space'>>) => string
  toCODERepository: (args: RequiredField<Pick<CODEProps, 'repoPath' | 'gitRef' | 'resourcePath'>, 'repoPath'>) => string
  toCODEFileEdit: (args: Required<Pick<CODEProps, 'repoPath' | 'gitRef' | 'resourcePath'>>) => string
  toCODECommits: (args: Required<Pick<CODEProps, 'repoPath' | 'commitRef'>>) => string
  toCODECommit: (args: Required<Pick<CODEProps, 'repoPath' | 'commitRef'>>) => string
  toCODEPullRequests: (args: Required<Pick<CODEProps, 'repoPath'>>) => string
  toCODEPullRequest: (
    args: RequiredField<
      Pick<CODEProps, 'repoPath' | 'pullRequestId' | 'pullRequestSection' | 'commitSHA'>,
      'repoPath' | 'pullRequestId'
    >
  ) => string
  toCODECompare: (args: Required<Pick<CODEProps, 'repoPath' | 'diffRefs'>>) => string
  toCODEBranches: (args: Required<Pick<CODEProps, 'repoPath'>>) => string
  toCODETags: (args: Required<Pick<CODEProps, 'repoPath'>>) => string
  toCODEWebhooks: (args: Required<Pick<CODEProps, 'repoPath'>>) => string
  toCODEWebhookNew: (args: Required<Pick<CODEProps, 'repoPath'>>) => string
  toCODEWebhookDetails: (args: Required<Pick<CODEProps, 'repoPath' | 'webhookId'>>) => string
  toCODESettings: (
    args: RequiredField<Pick<CODEProps, 'repoPath' | 'settingSection' | 'ruleId' | 'settingSectionMode'>, 'repoPath'>
  ) => string
  toCODESpaceSearch: (args: Required<Pick<CODEProps, 'space'>>) => string
  toCODERepositorySearch: (args: Required<Pick<CODEProps, 'repoPath'>>) => string
  toCODESemanticSearch: (args: Required<Pick<CODEProps, 'repoPath'>>) => string
  toCODEExecutions: (args: Required<Pick<CODEProps, 'repoPath' | 'pipeline'>>) => string
  toCODEExecution: (args: Required<Pick<CODEProps, 'repoPath' | 'pipeline' | 'execution'>>) => string
  toCODESecret: (args: Required<Pick<CODEProps, 'space' | 'secret'>>) => string
}

/**
 * NOTE: NEVER IMPORT AND USE THIS ROUTES EXPORT DIRECTLY IN CODE.
 *
 * routes is used to created URLs in standalone version. Instead, use
 * the `routes` from AppContext which is mapped to this export in standalone
 * version or Harness Platform routes which is passed from Harness Platform UI.
 *
 * Correct usage: const { routes } = useAppContext()
 */
export const routes: CODERoutes = {
  toSignIn: (): string => '/signin',
  toRegister: (): string => '/register',

  toCODEHome: () => `/`,

  toCODESpaceAccessControl: ({ space }) => `/access-control/${space}`,
  toCODESpaceSettings: ({ space }) => `/settings/${space}`,
  toCODEPipelines: ({ repoPath }) => `/${repoPath}/pipelines`,
  toCODEPipelineEdit: ({ repoPath, pipeline }) => `/${repoPath}/pipelines/${pipeline}/edit`,
  toCODEPipelineSettings: ({ repoPath, pipeline }) => `/${repoPath}/pipelines/${pipeline}/triggers`,
  toCODESecrets: ({ space }) => `/secrets/${space}`,

  toCODEGlobalSettings: () => '/settings',
  toCODEUsers: () => '/users',
  toCODEUserProfile: () => '/profile',
  toCODEUserChangePassword: () => '/change-password',

  toCODERepositories: ({ space }) => `/spaces/${space}`,
  toCODERepository: ({ repoPath, gitRef, resourcePath }) =>
    `/${repoPath}${gitRef ? '/files/' + gitRef : ''}${resourcePath ? '/~/' + resourcePath : ''}`,
  toCODEFileEdit: ({
    repoPath,
    gitRef,
    resourcePath
  }: RequiredField<Pick<CODEProps, 'repoPath' | 'gitRef' | 'resourcePath'>, 'repoPath' | 'gitRef'>) =>
    `/${repoPath}/edit/${gitRef}/~/${resourcePath || ''}`,

  toCODECommits: ({ repoPath, commitRef }) => `/${repoPath}/commits/${commitRef}`,
  toCODECommit: ({ repoPath, commitRef }) => `/${repoPath}/commit/${commitRef}`,
  toCODEPullRequests: ({ repoPath }) => `/${repoPath}/pulls`,
  toCODEPullRequest: ({ repoPath, pullRequestId, pullRequestSection, commitSHA }) =>
    `/${repoPath}/pulls/${pullRequestId}${pullRequestSection ? '/' + pullRequestSection : ''}${
      commitSHA ? '/' + commitSHA : ''
    }`,
  toCODECompare: ({ repoPath, diffRefs }) => `/${repoPath}/pulls/compare/${diffRefs}`,
  toCODEBranches: ({ repoPath }) => `/${repoPath}/branches`,
  toCODETags: ({ repoPath }) => `/${repoPath}/tags`,
  toCODESettings: ({ repoPath, settingSection, ruleId, settingSectionMode }) =>
    `/${repoPath}/settings${settingSection ? '/' + settingSection : ''}${ruleId ? '/' + ruleId : ''}${
      settingSectionMode ? '/' + settingSectionMode : ''
    }`,
  toCODESpaceSearch: ({ space }) => `/${space}/search`,
  toCODERepositorySearch: ({ repoPath }) => `/${repoPath}/search`,
  toCODESemanticSearch: ({ repoPath }) => `/${repoPath}/search/semantic`,
  toCODEWebhooks: ({ repoPath }) => `/${repoPath}/webhooks`,
  toCODEWebhookNew: ({ repoPath }) => `/${repoPath}/webhooks/new`,
  toCODEWebhookDetails: ({ repoPath, webhookId }) => `/${repoPath}/webhook/${webhookId}`,

  toCODEExecutions: ({ repoPath, pipeline }) => `/${repoPath}/pipelines/${pipeline}`,
  toCODEExecution: ({ repoPath, pipeline, execution }) => `/${repoPath}/pipelines/${pipeline}/execution/${execution}`,
  toCODESecret: ({ space, secret }) => `/secrets/${space}/secret/${secret}`,
  ...cdeRoutes
}
