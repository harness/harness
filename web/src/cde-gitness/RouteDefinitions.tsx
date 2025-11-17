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

export interface CDEProps {
  space?: string
  gitspaceId?: string
  accountId?: string
  infraprovider_identifier?: string
  provider?: string
  aitaskId?: string
}

export const pathProps: Readonly<Required<CDEProps>> = {
  space: ':space*',
  gitspaceId: ':gitspaceId',
  accountId: ':accountId',
  infraprovider_identifier: ':infraprovider_identifier',
  provider: ':provider',
  aitaskId: ':taskId'
}

export interface CDERoutes {
  toCDEGitspaces: (args: Required<Pick<CDEProps, 'space'>>) => string
  toCDEAITasks: (args: Required<Pick<CDEProps, 'space'>>) => string
  toCDEGitspaceDetail: (args: Required<Pick<CDEProps, 'space' | 'gitspaceId'>>) => string
  toCDEGitspacesCreate: (args: Required<Pick<CDEProps, 'space'>>) => string
  toCDEAITaskCreate: (args: Required<Pick<CDEProps, 'space'>>) => string
  toCDEGitspacesEdit: (args: Required<Pick<CDEProps, 'space' | 'gitspaceId'>>) => string
  toCDEAITaskDetail: (args: Required<Pick<CDEProps, 'space' | 'aitaskId'>>) => string
  toCDEGitspaceInfra: (args: Required<Pick<CDEProps, 'accountId'>>) => string
  toCDEUsageDashboard: (args: Required<Pick<CDEProps, 'accountId'>>) => string
  toCDEInfraConfigure: (args: Required<Pick<CDEProps, 'accountId' | 'provider'>>) => string
  toModuleRoute: (args: Required<Pick<CDEProps, 'accountId'>>) => string
  toCDEInfraConfigureDetail: (
    args: Required<Pick<CDEProps, 'accountId' | 'infraprovider_identifier' | 'provider'>>
  ) => string
  toCDEInfraConfigureDetailDownload: (
    args: Required<Pick<CDEProps, 'accountId' | 'infraprovider_identifier' | 'provider'>>
  ) => string
}

export const routes: CDERoutes = {
  toCDEGitspaces: ({ space }) => `/${space}/gitspaces`,
  toCDEAITasks: ({ space }) => `/${space}/aitasks`,
  toCDEGitspaceDetail: ({ space, gitspaceId }) => `/${space}/gitspaces/${gitspaceId}`,
  toCDEAITaskDetail: ({ space, aitaskId }) => `/${space}/aitasks/${aitaskId}`,
  toCDEGitspacesCreate: ({ space }) => `/${space}/gitspaces/create`,
  toCDEAITaskCreate: ({ space }) => `/${space}/aitasks/create`,
  toCDEGitspacesEdit: ({ space, gitspaceId }) => `/${space}/gitspaces/edit/${gitspaceId}`,
  toCDEGitspaceInfra: ({ accountId }) => `/account/${accountId}/module/cde/gitspace-infrastructure`,
  toCDEUsageDashboard: ({ accountId }) => `/account/${accountId}/module/cde/usage-dashboard`,
  toCDEInfraConfigure: ({ accountId, provider }) =>
    `/account/${accountId}/module/cde/gitspace-infrastructure/configure/${provider}`,
  toModuleRoute: ({ accountId }) => `/account/${accountId}/module/cde`,
  toCDEInfraConfigureDetail: ({ accountId, infraprovider_identifier, provider }) =>
    `/account/${accountId}/module/cde/gitspace-infrastructure/configure/${provider}/${infraprovider_identifier}`,
  toCDEInfraConfigureDetailDownload: ({ accountId, infraprovider_identifier, provider }) =>
    `/account/${accountId}/module/cde/gitspace-infrastructure/configure/${provider}/${infraprovider_identifier}/download`
}
