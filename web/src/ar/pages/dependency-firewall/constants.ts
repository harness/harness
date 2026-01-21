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

import type { Parent } from '@ar/common/types'
import type { StringKeys } from '@ar/frameworks/strings'
import type { FeatureFlags } from '@ar/MFEAppTypes'

export enum DependencyFirewallTab {
  VIOLATIONS = 'violations',
  EXCEPTIONS = 'exceptions'
}

interface DependencyFirewallTabSpec {
  label: StringKeys
  value: DependencyFirewallTab
  featureFlag?: FeatureFlags
  supportActions?: boolean
  parent?: Parent
  disabled?: boolean
}

export const DependencyFirewallTabs: DependencyFirewallTabSpec[] = [
  {
    label: 'dependencyFirewall.tabs.violations',
    value: DependencyFirewallTab.VIOLATIONS
  },
  {
    label: 'dependencyFirewall.tabs.exceptions',
    value: DependencyFirewallTab.EXCEPTIONS,
    disabled: true
  }
]
