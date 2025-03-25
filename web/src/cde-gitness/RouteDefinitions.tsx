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
}

export const pathProps: Readonly<Required<CDEProps>> = {
  space: ':space*',
  gitspaceId: ':gitspaceId',
  accountId: ':accountId'
}

export interface CDERoutes {
  toCDEGitspaces: (args: Required<Pick<CDEProps, 'space'>>) => string
  toCDEGitspaceDetail: (args: Required<Pick<CDEProps, 'space' | 'gitspaceId'>>) => string
  toCDEGitspacesCreate: (args: Required<Pick<CDEProps, 'space'>>) => string
  toCDEGitspacesEdit: (args: Required<Pick<CDEProps, 'space' | 'gitspaceId'>>) => string
  toCDEGitspaceInfra: (args: Required<Pick<CDEProps, 'accountId'>>) => string
  toCDEInfraConfigure: (args: Required<Pick<CDEProps, 'accountId'>>) => string
  toModuleRoute: (args: Required<Pick<CDEProps, 'accountId'>>) => string
}

export const routes: CDERoutes = {
  toCDEGitspaces: ({ space }) => `/${space}/gitspaces`,
  toCDEGitspaceDetail: ({ space, gitspaceId }) => `/${space}/gitspaces/${gitspaceId}`,
  toCDEGitspacesCreate: ({ space }) => `/${space}/gitspaces/create`,
  toCDEGitspacesEdit: ({ space, gitspaceId }) => `/${space}/gitspaces/edit/${gitspaceId}`,
  toCDEGitspaceInfra: ({ accountId }) => `/account/${accountId}/module/cde/gitspace-infrastructure`,
  toCDEInfraConfigure: ({ accountId }) => `/account/${accountId}/module/cde/gitspace-infrastructure/configure`,
  toModuleRoute: ({ accountId }) => `/account/${accountId}/module/cde`
}
