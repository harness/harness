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

import type { StringKeys } from '@ar/frameworks/strings'

export enum ManageRegistriesDetailsTab {
  LABELS = 'labels',
  CLEANUP_POLICIES = 'cleanup_policies'
}

interface ManageRegistriesDetailsTabSpec {
  label: StringKeys
  value: ManageRegistriesDetailsTab
  disabled?: boolean
  tooltip?: StringKeys
}

export const ManageRegistriesDetailsTabs: ManageRegistriesDetailsTabSpec[] = [
  {
    label: 'manageRegistries.tabs.labels',
    value: ManageRegistriesDetailsTab.LABELS
  },
  {
    label: 'manageRegistries.tabs.cleanup_policies',
    value: ManageRegistriesDetailsTab.CLEANUP_POLICIES,
    disabled: true,
    tooltip: 'manageRegistries.comingSoon'
  }
]
