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
import type { Scanner } from '@harnessio/react-har-service-client'

import { FeatureFlags } from '@ar/MFEAppTypes'
import type { StringKeys } from '@ar/frameworks/strings'
import { RepositoryListViewTypeEnum } from '@ar/contexts/AppStoreContext'
import { RepositoryConfigType, RepositoryPackageType, Scanners } from '@ar/common/types'

export const POLICY_TYPE = 'securityTests'
export const POLICY_ACTION = 'onstep'

export enum RepositoryDetailsTab {
  PACKAGES = 'packages',
  DATASETS = 'datasets',
  MODELS = 'models',
  CONFIGURATION = 'configuration',
  WEBHOOKS = 'webhooks'
}

export enum LocalArtifactType {
  ARTIFACTS = 'artifacts',
  DATASET = 'dataset',
  MODEL = 'model'
}

export interface ScannerConfigSpec {
  icon: IconName
  label: string
  value: Scanner['name']
  tooltipId?: string
}

export const ContainerScannerConfig: Record<Scanners, ScannerConfigSpec> = {
  AQUA_TRIVY: {
    icon: 'AquaTrivy',
    label: 'AquaTrivy',
    value: 'AQUA_TRIVY'
  },
  GRYPE: {
    icon: 'anchore-grype',
    label: 'Grype',
    value: 'GRYPE'
  }
}

interface RepositoryDetailsTabSpec {
  label: StringKeys
  value: RepositoryDetailsTab
  artifactType?: LocalArtifactType
  packageType?: RepositoryPackageType
  type?: RepositoryConfigType
  mode?: RepositoryListViewTypeEnum
  featureFlag?: FeatureFlags
}

export const RepositoryDetailsTabs: RepositoryDetailsTabSpec[] = [
  {
    label: 'repositoryDetails.tabs.packages',
    value: RepositoryDetailsTab.PACKAGES,
    mode: RepositoryListViewTypeEnum.LIST,
    artifactType: LocalArtifactType.ARTIFACTS
  },
  {
    label: 'repositoryDetails.tabs.datasets',
    value: RepositoryDetailsTab.DATASETS,
    mode: RepositoryListViewTypeEnum.LIST,
    artifactType: LocalArtifactType.DATASET
  },
  {
    label: 'repositoryDetails.tabs.models',
    value: RepositoryDetailsTab.MODELS,
    mode: RepositoryListViewTypeEnum.LIST,
    artifactType: LocalArtifactType.MODEL
  },
  {
    label: 'repositoryDetails.tabs.configuration',
    value: RepositoryDetailsTab.CONFIGURATION
  },
  {
    label: 'repositoryDetails.tabs.webhooks',
    value: RepositoryDetailsTab.WEBHOOKS,
    featureFlag: FeatureFlags.HAR_TRIGGERS,
    type: RepositoryConfigType.VIRTUAL
  }
]
