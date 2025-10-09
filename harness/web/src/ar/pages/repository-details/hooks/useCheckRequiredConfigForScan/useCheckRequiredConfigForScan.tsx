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

import { useGetOrgScopedProjectQuery } from '@harnessio/react-ng-manager-client'

import { useAppStore, useLicenseStore } from '@ar/hooks'
import { LICENSE_STATE_VALUES } from '@ar/common/LicenseTypes'
import { DEFAULT_ORG, DEFAULT_PROJECT } from '@ar/constants'

export default function useCheckRequiredConfigForScan(): {
  hasRequiredLicense: boolean
  hasRequiredProjectConfig: boolean
  hasRequiredConfig: boolean
  orgIdentifier: string
} {
  const { scope } = useAppStore()
  const { orgIdentifier, projectIdentifier } = scope
  const { SSCA_LICENSE_STATE, STO_LICENSE_STATE } = useLicenseStore()
  const hasRequiredLicense =
    SSCA_LICENSE_STATE === LICENSE_STATE_VALUES.ACTIVE && STO_LICENSE_STATE === LICENSE_STATE_VALUES.ACTIVE

  const { isFetching, error } = useGetOrgScopedProjectQuery(
    {
      org: orgIdentifier ?? DEFAULT_ORG,
      project: projectIdentifier ?? DEFAULT_PROJECT
    },
    {
      enabled: !projectIdentifier
    }
  )
  const hasRequiredProjectConfig = !isFetching && !error

  const hasRequiredConfig = hasRequiredLicense && hasRequiredProjectConfig

  return {
    hasRequiredLicense,
    hasRequiredProjectConfig,
    hasRequiredConfig,
    orgIdentifier: orgIdentifier ?? DEFAULT_ORG
  }
}
