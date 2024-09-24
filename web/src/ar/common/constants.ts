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

import type { IconName } from '@harnessio/icons'
import type { StringsMap } from '@ar/frameworks/strings'
import { EnvironmentType, RepositoryPackageType } from './types'

export interface RepositoryTypeListItem {
  label: keyof StringsMap
  value: RepositoryPackageType
  icon: IconName
  disabled?: boolean
  tooltip?: string
}

export const RepositoryTypes: RepositoryTypeListItem[] = [
  {
    label: 'repositoryTypes.docker',
    value: RepositoryPackageType.DOCKER,
    icon: 'docker-step'
  },
  {
    label: 'repositoryTypes.helm',
    value: RepositoryPackageType.HELM,
    icon: 'service-helm'
  },
  {
    label: 'repositoryTypes.generic',
    value: RepositoryPackageType.GENERIC,
    icon: 'generic-repository-type',
    tooltip: 'Coming Soon!',
    disabled: true
  },
  {
    label: 'repositoryTypes.maven',
    value: RepositoryPackageType.MAVEN,
    icon: 'maven-repository-type',
    tooltip: 'Coming Soon!',
    disabled: true
  },
  {
    label: 'repositoryTypes.npm',
    value: RepositoryPackageType.NPM,
    icon: 'npm-repository-type',
    tooltip: 'Coming Soon!',
    disabled: true
  },
  {
    label: 'repositoryTypes.gradle',
    value: RepositoryPackageType.GRADLE,
    icon: 'gradle-repository-type',
    tooltip: 'Coming Soon!',
    disabled: true
  },
  {
    label: 'repositoryTypes.pypi',
    value: RepositoryPackageType.PYPI,
    icon: 'python',
    tooltip: 'Coming Soon!',
    disabled: true
  },
  {
    label: 'repositoryTypes.nuget',
    value: RepositoryPackageType.NUGET,
    icon: 'nuget-repository-type',
    tooltip: 'Coming Soon!',
    disabled: true
  }
]

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
