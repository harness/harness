/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import type { StringsMap } from '@ar/frameworks/strings'
import { EntityScope, EnvironmentType, RepositoryConfigType, RepositoryScopeType } from './types'

interface EnvironmentTypeListItem {
  label: keyof StringsMap
  value: EnvironmentType
  disabled?: boolean
}

export const EnvironmentTypeList: EnvironmentTypeListItem[] = [
  {
    label: 'prod',
    value: EnvironmentType.Prod
  },
  {
    label: 'nonProd',
    value: EnvironmentType.NonProd
  }
]

interface RepositoryConfigTypesListItem {
  label: keyof StringsMap
  value: RepositoryConfigType
  disabled?: boolean
  tooltip?: string
}

export const RepositoryConfigTypes: RepositoryConfigTypesListItem[] = [
  {
    label: 'repositoryList.artifactRegistry.label',
    value: RepositoryConfigType.VIRTUAL
  },
  {
    label: 'repositoryList.upstreamProxy.label',
    value: RepositoryConfigType.UPSTREAM
  }
]

interface RepositoryScopeTypesListItem {
  label: keyof StringsMap
  value: RepositoryScopeType
  scope: EntityScope
}

export const RepositoryScopeTypes: RepositoryScopeTypesListItem[] = [
  {
    label: 'repositoryList.scope.accountOnly',
    value: RepositoryScopeType.NONE,
    scope: EntityScope.ACCOUNT
  },
  {
    label: 'repositoryList.scope.orgOnly',
    value: RepositoryScopeType.NONE,
    scope: EntityScope.ORG
  },
  {
    label: 'repositoryList.scope.accountRecursive',
    value: RepositoryScopeType.DESCENDANTS,
    scope: EntityScope.ACCOUNT
  },
  {
    label: 'repositoryList.scope.orgRecursive',
    value: RepositoryScopeType.DESCENDANTS,
    scope: EntityScope.ORG
  }
]
