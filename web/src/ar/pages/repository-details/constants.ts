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

import type { Scanners } from '@ar/common/types'

export enum RepositoryDetailsTab {
  PACKAGES = 'packages',
  CONFIGURATION = 'configuration'
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
