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
import { FeatureFlags } from '@ar/MFEAppTypes'
import type { StringsMap } from '@ar/frameworks/strings'
import { RepositoryPackageType } from '@ar/common/types'
import { ThumbnailTagEnum } from '@ar/components/Tag/ThumbnailTags'
import { useFeatureFlags } from './useFeatureFlag'

export interface RepositoryTypeListItem {
  label: keyof StringsMap
  value: RepositoryPackageType
  icon: IconName
  disabled?: boolean
  tooltip?: string
  featureFlag?: FeatureFlags
  tag?: ThumbnailTagEnum
}

export const useGetRepositoryTypes = (): RepositoryTypeListItem[] => {
  const featureFlags = useFeatureFlags()

  return RepositoryTypes.map(repo => {
    if (repo.disabled && repo.featureFlag && featureFlags[repo.featureFlag]) {
      return {
        ...repo,
        disabled: false,
        tag: repo.tag === ThumbnailTagEnum.ComingSoon ? ThumbnailTagEnum.Beta : repo.tag,
        tooltip: undefined
      }
    }
    return repo
  })
}

const RepositoryTypes: RepositoryTypeListItem[] = [
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
    icon: 'generic-repository-type'
  },
  {
    label: 'repositoryTypes.maven',
    value: RepositoryPackageType.MAVEN,
    icon: 'maven-repository-type'
  },
  {
    label: 'repositoryTypes.pypi',
    value: RepositoryPackageType.PYTHON,
    icon: 'python'
  },
  {
    label: 'repositoryTypes.npm',
    value: RepositoryPackageType.NPM,
    icon: 'npm-repository-type'
  },
  {
    label: 'repositoryTypes.nuget',
    value: RepositoryPackageType.NUGET,
    icon: 'nuget-repository-type'
  },
  {
    label: 'repositoryTypes.rpm',
    value: RepositoryPackageType.RPM,
    icon: 'red-hat-logo',
    tooltip: 'Coming Soon!',
    disabled: true,
    tag: ThumbnailTagEnum.ComingSoon,
    featureFlag: FeatureFlags.HAR_RPM_PACKAGE_TYPE_ENABLED
  },
  {
    label: 'repositoryTypes.cargo',
    value: RepositoryPackageType.CARGO,
    icon: 'rust-logo',
    tooltip: 'Coming Soon!',
    disabled: true,
    tag: ThumbnailTagEnum.ComingSoon,
    featureFlag: FeatureFlags.HAR_CARGO_PACKAGE_TYPE_ENABLED
  },
  {
    label: 'repositoryTypes.go',
    value: RepositoryPackageType.GO,
    icon: 'go-logo',
    tooltip: 'Coming Soon!',
    disabled: true,
    tag: ThumbnailTagEnum.ComingSoon
  },
  {
    label: 'repositoryTypes.debian',
    value: RepositoryPackageType.DEBIAN,
    icon: 'debian-logo',
    tooltip: 'Coming Soon!',
    disabled: true,
    tag: ThumbnailTagEnum.ComingSoon
  },
  {
    label: 'repositoryTypes.alpine',
    value: RepositoryPackageType.ALPINE,
    icon: 'alpine-logo',
    tooltip: 'Coming Soon!',
    disabled: true,
    tag: ThumbnailTagEnum.ComingSoon
  }
]
