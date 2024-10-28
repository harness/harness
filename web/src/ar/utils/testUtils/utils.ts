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

import { createContext } from 'react'
import { noop } from 'lodash-es'

import RbacButton from '@ar/__mocks__/components/RbacButton'
import RbacMenuItem from '@ar/__mocks__/components/RbacMenuItem'
import NGBreadcrumbs from '@ar/__mocks__/components/NGBreadcrumbs'
import DependencyView from '@ar/__mocks__/components/DependencyView'
import SecretFormInput from '@ar/__mocks__/components/SecretFormInput'
import VulnerabilityView from '@ar/__mocks__/components/VulnerabilityView'
import { usePreferenceStore } from '@ar/__mocks__/contexts/PreferenceStoreContext'
import {
  ModalProvider,
  useConfirmationDialog,
  useDefaultPaginationProps,
  useModalHook,
  useQueryParams,
  useQueryParamsOptions,
  useUpdateQueryParams
} from '@ar/__mocks__/hooks'

import { LICENSE_STATE_VALUES } from '@ar/common/LicenseTypes'
import type { ParentProviderProps } from '@ar/contexts/ParentProvider'
import { getApiBaseUrl } from '@ar/__mocks__/utils/getApiBaseUrl'
import getCustomHeaders from '@ar/__mocks__/utils/getCustomHeaders'

export const MockTestUtils: {
  hooks?: ParentProviderProps['hooks']
  components?: ParentProviderProps['components']
  utils?: ParentProviderProps['utils']
} = {}

MockTestUtils.hooks = {
  useDocumentTitle: () => ({ updateTitle: () => void 0 }),
  useLogout: () => ({ forceLogout: () => void 0 }),
  usePermission: () => [true],
  useQueryParams,
  useUpdateQueryParams,
  useQueryParamsOptions,
  useDefaultPaginationProps,
  usePreferenceStore,
  useModalHook,
  useConfirmationDialog
}

MockTestUtils.components = {
  RbacButton,
  NGBreadcrumbs,
  RbacMenuItem,
  SecretFormInput,
  VulnerabilityView,
  DependencyView,
  ModalProvider
}

MockTestUtils.utils = {
  getCustomHeaders,
  getApiBaseUrl,
  generateToken: async () => ''
}

export const MockLicenseContext = createContext({
  versionMap: {},
  licenseInformation: {},
  STO_LICENSE_STATE: LICENSE_STATE_VALUES.ACTIVE,
  SSCA_LICENSE_STATE: LICENSE_STATE_VALUES.ACTIVE,
  CI_LICENSE_STATE: LICENSE_STATE_VALUES.ACTIVE,
  CD_LICENSE_STATE: LICENSE_STATE_VALUES.ACTIVE
})

export const MockParentAppStoreContext = createContext({
  featureFlags: {},
  updateAppStore: noop
})

export const MockPermissionsContext = createContext({})
export const MockTooltipContext = createContext({})
export const MockTokenContext = createContext({})

export const getTableRow = (row: number) => {
  return document.querySelector(`div[class*="TableV2--body--"] [class*="TableV2--row"]:nth-child(${row})`)
}

export const getTableColumn = (row: number, column: number): HTMLDivElement | null => {
  return document.querySelector(
    `div[class*="TableV2--body--"] [class*="TableV2--row"]:nth-child(${row}) [class*="TableV2--cell--"]:nth-child(${column})`
  )
}
