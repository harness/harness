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

import type { IconName, IconProps } from '@harnessio/icons'
import type { StringsMap } from '@ar/frameworks/strings'

export enum VersionDetailsTab {
  OVERVIEW = 'overview',
  ARTIFACT_DETAILS = 'artifact_details',
  SUPPLY_CHAIN = 'supply_chain',
  SECURITY_TESTS = 'security_tests',
  DEPLOYMENTS = 'deployments',
  CODE = 'code',
  OSS = 'OSS' // added for open source system view
}

interface VersionDetailsTabListItem {
  label: keyof StringsMap
  value: VersionDetailsTab
  icon: IconName
  iconProps: Omit<IconProps, 'name'>
}

export const VersionDetailsTabList: VersionDetailsTabListItem[] = [
  {
    label: 'versionDetails.tabs.overview',
    value: VersionDetailsTab.OVERVIEW,
    icon: 'store-artifact-bundle',
    iconProps: {
      size: 18
    }
  },
  {
    label: 'versionDetails.tabs.artifactDetails',
    value: VersionDetailsTab.ARTIFACT_DETAILS,
    icon: 'layers',
    iconProps: {
      size: 16
    }
  },
  {
    label: 'versionDetails.tabs.supplyChain',
    value: VersionDetailsTab.SUPPLY_CHAIN,
    icon: 'ssca-main',
    iconProps: {
      size: 16
    }
  },
  {
    label: 'versionDetails.tabs.securityTests',
    value: VersionDetailsTab.SECURITY_TESTS,
    icon: 'sto-color-filled',
    iconProps: {
      size: 16
    }
  },
  {
    label: 'versionDetails.tabs.deployments',
    value: VersionDetailsTab.DEPLOYMENTS,
    icon: 'cd-main',
    iconProps: {
      size: 16
    }
  },
  {
    label: 'versionDetails.tabs.code',
    value: VersionDetailsTab.CODE,
    icon: 'code',
    iconProps: {
      size: 16
    }
  }
]
