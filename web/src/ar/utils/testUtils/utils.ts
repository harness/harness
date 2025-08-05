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
import userEvent from '@testing-library/user-event'
import { getByText, waitFor } from '@testing-library/react'

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
  generateToken: jest.fn(),
  getRouteToPipelineExecutionView: jest.fn(),
  routeToRegistryDetails: jest.fn()
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

export const getTableHeaderColumn = (column: number) => {
  return document.querySelector(`div[class*="TableV2--header--"] [class*="TableV2--cell"]:nth-child(${column})`)
}

export const getTableRow = (row: number, container: Document | Element = document): HTMLDivElement | null => {
  return container.querySelector(`div[class*="TableV2--body--"] [class*="TableV2--row"]:nth-child(${row})`)
}

export const getTableColumn = (
  row: number,
  column: number,
  container: Document | Element = document
): HTMLDivElement | null => {
  const rowElement = getTableRow(row, container)
  if (rowElement) {
    return rowElement.querySelector(`[class*="TableV2--cell--"]:nth-child(${column})`)
  }
  return null
}

export const testSelectChange = async (
  element: HTMLElement,
  optionToSelect: string,
  selectedOption?: string,
  popoverClass = 'bp3-select-popover',
  alreadyOpenedDialogs = 0
): Promise<void> => {
  await userEvent.click(element)
  const dialogs = document.getElementsByClassName(popoverClass)
  await waitFor(() => expect(dialogs).toHaveLength(alreadyOpenedDialogs + 1))
  const selectPopover = dialogs[alreadyOpenedDialogs] as HTMLElement

  if (selectedOption) {
    const selectedOptionRef = getByText(selectPopover, selectedOption)
    expect(selectedOptionRef).toBeInTheDocument()
  }

  const optionToSelectRef = getByText(selectPopover, optionToSelect)
  expect(optionToSelectRef).toBeInTheDocument()
  await userEvent.click(optionToSelectRef)
  await waitFor(() => expect(dialogs).toHaveLength(alreadyOpenedDialogs))
}

export const testMultiSelectChange = async (
  element: HTMLElement,
  optionToSelect: string,
  selectedOption?: string,
  popoverClass = 'bp3-popover',
  alreadyOpenedDialogs = 0
): Promise<void> => {
  await userEvent.click(element)
  const dialogs = document.getElementsByClassName(popoverClass)
  await waitFor(() => expect(dialogs).toHaveLength(alreadyOpenedDialogs + 1))
  const selectPopover = dialogs[alreadyOpenedDialogs] as HTMLElement

  if (selectedOption) {
    const selectedOptionRef = getByText(selectPopover, selectedOption)
    expect(selectedOptionRef).toBeInTheDocument()
  }

  const optionToSelectRef = getByText(selectPopover, optionToSelect)
  expect(optionToSelectRef).toBeInTheDocument()
  await userEvent.click(optionToSelectRef)
  await userEvent.click(document.getElementsByClassName('bp3-popover-backdrop')[0]!)
  await waitFor(() => expect(dialogs).toHaveLength(alreadyOpenedDialogs))
}
