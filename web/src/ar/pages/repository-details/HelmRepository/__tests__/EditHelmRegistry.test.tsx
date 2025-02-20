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

import React from 'react'
import userEvent from '@testing-library/user-event'
import { fireEvent, getByTestId, getByText, queryByTestId, render, waitFor } from '@testing-library/react'
import { useGetClientSetupDetailsQuery, useGetRegistryQuery } from '@harnessio/react-har-service-client'

import { DEFAULT_DATE_TIME_FORMAT } from '@ar/constants'
import { getReadableDateTime } from '@ar/common/dateUtils'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import { queryByNameAttribute } from 'utils/test/testUtils'

import RepositoryDetailsPage from '../../RepositoryDetailsPage'
import {
  MockGetHelmArtifactsByRegistryResponse,
  MockGetHelmRegistryResponseWithAllData,
  MockGetHelmSetupClientOnRegistryConfigPageResponse
} from './__mockData__'
import '../../RepositoryFactory'

const modifyRepository = jest.fn().mockImplementation(
  () =>
    new Promise(onSuccess => {
      onSuccess({ content: { status: 'SUCCESS' } })
    })
)

const mockHistoryPush = jest.fn()
// eslint-disable-next-line jest-no-mock
jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useHistory: () => ({
    push: mockHistoryPush
  })
}))

jest.mock('@harnessio/react-har-service-client', () => ({
  useGetRegistryQuery: jest.fn().mockImplementation(() => ({
    isFetching: false,
    refetch: jest.fn(),
    error: false,
    data: MockGetHelmRegistryResponseWithAllData
  })),
  useGetAllArtifactsByRegistryQuery: jest.fn().mockImplementation(() => ({
    isFetching: false,
    refetch: jest.fn(),
    error: false,
    data: MockGetHelmArtifactsByRegistryResponse
  })),
  useGetClientSetupDetailsQuery: jest.fn().mockImplementation(() => ({
    isFetching: false,
    refetch: jest.fn(),
    error: false,
    data: MockGetHelmSetupClientOnRegistryConfigPageResponse
  })),
  useDeleteRegistryMutation: jest.fn().mockImplementation(() => ({
    isLoading: false,
    mutateAsync: jest.fn()
  })),
  useModifyRegistryMutation: jest.fn().mockImplementation(() => ({
    isLoading: false,
    mutateAsync: modifyRepository
  })),
  useGetAllRegistriesQuery: jest.fn().mockImplementation(() => ({
    isFetching: false,
    data: { content: { data: { registries: [] }, status: 'SUCCESS' } },
    refetch: jest.fn(),
    error: null
  }))
}))

describe('Verify header section for helm artifact registry', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  test('Verify breadcrumbs', async () => {
    const { container } = render(
      <ArTestWrapper>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )

    const pageHeader = container.querySelector('div[data-testid=page-header]')
    expect(pageHeader).toBeInTheDocument()

    const breadcrumbsSection = pageHeader?.querySelector('div[class*=PageHeader--breadcrumbsDiv--]')
    expect(breadcrumbsSection).toBeInTheDocument()

    expect(breadcrumbsSection).toHaveTextContent('breadcrumbs.repositories')
  })

  test('Verify registry icon, registry name, tag, lables, description and last updated', async () => {
    const { container } = render(
      <ArTestWrapper>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )
    const pageHeader = getByTestId(container, 'registry-header-container')
    expect(pageHeader).toBeInTheDocument()

    expect(pageHeader?.querySelector('span[data-icon=service-helm]')).toBeInTheDocument()
    const data = MockGetHelmRegistryResponseWithAllData.content.data

    const title = getByTestId(container, 'registry-title')
    expect(title).toHaveTextContent(data.identifier)

    const description = getByTestId(container, 'registry-description')
    expect(description).toHaveTextContent(data.description)

    expect(pageHeader?.querySelector('svg[data-icon=tag]')).toBeInTheDocument()

    const lastModifiedAt = getByTestId(container, 'registry-last-modified-at')
    expect(lastModifiedAt).toHaveTextContent(getReadableDateTime(Number(data.modifiedAt), DEFAULT_DATE_TIME_FORMAT))
  })

  test('Verify registry setup client action', async () => {
    const { container } = render(
      <ArTestWrapper>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )
    const pageHeader = getByTestId(container, 'registry-header-container')
    const setupClientBtn = pageHeader.querySelector('button[aria-label="actions.setupClient"]')
    expect(setupClientBtn).toBeInTheDocument()
    await userEvent.click(setupClientBtn!)

    await waitFor(() => {
      expect(useGetClientSetupDetailsQuery).toHaveBeenLastCalledWith({
        queryParams: { artifact: undefined, version: undefined },
        registry_ref: 'undefined/helm-repo/+'
      })
    })
  })

  test('Verify other registry actions', async () => {
    const { container } = render(
      <ArTestWrapper>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )
    const pageHeader = getByTestId(container, 'registry-header-container')
    const actions3DotsBtn = pageHeader.querySelector('span[data-icon=Options')
    expect(actions3DotsBtn).toBeInTheDocument()

    await userEvent.click(actions3DotsBtn!)
    const dialogs = document.getElementsByClassName('bp3-popover')
    await waitFor(() => expect(dialogs).toHaveLength(1))
    const selectPopover = dialogs[0] as HTMLElement

    const items = selectPopover.getElementsByClassName('bp3-menu-item')
    for (let idx = 0; idx < items.length; idx++) {
      const actionItem = items[idx]
      expect(actionItem.querySelector('span[data-icon=code-delete]')).toBeInTheDocument()
      expect(actionItem).toHaveTextContent('actions.delete')
    }
  })

  test('Verify tab selection status', async () => {
    const { container } = render(
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/:tab"
        pathParams={{ repositoryIdentifier: 'abcd', tab: 'packages' }}>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )

    const tabList = container.querySelector('div[role=tablist]')
    expect(tabList).toBeInTheDocument()

    const artifactsTab = tabList?.querySelector('div[data-tab-id=packages][aria-selected=true]')
    expect(artifactsTab).toBeInTheDocument()

    const configurationTab = tabList?.querySelector('div[data-tab-id=configuration][aria-selected=false]')
    expect(configurationTab).toBeInTheDocument()

    await userEvent.click(configurationTab!)
    expect(mockHistoryPush).toHaveBeenCalledWith('/registries/abcd/configuration')
  })
})

describe('Verify configuration form', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  test('should render form correctly with all data prefilled', async () => {
    const { container } = render(
      <ArTestWrapper path="/registries/abcd/:tab" pathParams={{ tab: 'configuration' }}>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )

    // Artifact registry defination section
    const registryDefinitionSection = getByTestId(container, 'registry-definition')
    const nameField = queryByNameAttribute('identifier', registryDefinitionSection)
    expect(nameField).toBeInTheDocument()
    expect(nameField).toBeDisabled()
    expect(nameField).toHaveAttribute('value', MockGetHelmRegistryResponseWithAllData.content.data.identifier)

    const descriptionField = queryByNameAttribute('description', registryDefinitionSection)
    expect(descriptionField).toBeInTheDocument()
    expect(descriptionField).not.toBeDisabled()
    expect(descriptionField).toHaveTextContent(MockGetHelmRegistryResponseWithAllData.content.data.description)

    const tags = registryDefinitionSection.querySelectorAll('div.bp3-tag-input-values .bp3-tag')
    tags.forEach((each, idx) => {
      expect(each).toHaveTextContent(MockGetHelmRegistryResponseWithAllData.content.data.labels[idx])
    })

    // Security scan section
    const securityScanSection = queryByTestId(container, 'security-section')
    expect(securityScanSection).toBeInTheDocument()

    // artifact filtering rules
    const filteringRulesSection = getByTestId(container, 'include-exclude-patterns-section')
    expect(filteringRulesSection).toBeInTheDocument()

    const allowedPatternsSection = filteringRulesSection.querySelectorAll('div.bp3-form-group')[0]
    const allowedPatterns = allowedPatternsSection.querySelectorAll('div.bp3-tag-input-values .bp3-tag')
    allowedPatterns.forEach((each, idx) => {
      expect(each).toHaveTextContent(MockGetHelmRegistryResponseWithAllData.content.data.allowedPattern[idx])
    })

    const blockedPatternsSection = filteringRulesSection.querySelectorAll('div.bp3-form-group')[1]
    const blockedPatterns = blockedPatternsSection.querySelectorAll('div.bp3-tag-input-values .bp3-tag')
    blockedPatterns.forEach((each, idx) => {
      expect(each).toHaveTextContent(MockGetHelmRegistryResponseWithAllData.content.data.blockedPattern[idx])
    })

    // upstream proxy section
    const upstreamProxySection = getByTestId(container, 'upstream-proxy-section')
    expect(upstreamProxySection).toBeInTheDocument()
    const selectedItemList = upstreamProxySection.querySelectorAll('ul[aria-label=orderable-list] .bp3-menu-item')
    selectedItemList.forEach((each, idx) => {
      expect(each).toHaveTextContent(MockGetHelmRegistryResponseWithAllData.content.data.config.upstreamProxies[idx])
    })

    // cleanup policy section
    const cleanupPoliciesSection = getByTestId(container, 'cleanup-policy-section')
    expect(cleanupPoliciesSection).toBeInTheDocument()
    const addCleanupPolicyBtn = cleanupPoliciesSection.querySelector(
      'a[role=button][aria-label="cleanupPolicy.addBtn"]'
    )
    expect(addCleanupPolicyBtn).toBeInTheDocument()
    expect(addCleanupPolicyBtn).toHaveAttribute('disabled', '')

    // action buttons
    const saveBtn = container.querySelector('button[aria-label=save]')
    expect(saveBtn).toBeDisabled()

    const discardBtn = container.querySelector('button[aria-label=discard]')
    expect(discardBtn).toBeDisabled()
  })

  test('should able to submit the form with updated data', async () => {
    const { container } = render(
      <ArTestWrapper path="/registries/abcd/:tab" pathParams={{ tab: 'configuration' }}>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )

    const descriptionField = queryByNameAttribute('description', container)
    fireEvent.change(descriptionField!, { target: { value: 'updated description' } })
    expect(descriptionField).toHaveTextContent('updated description')

    const saveBtn = container.querySelector('button[aria-label=save]')
    expect(saveBtn).not.toBeDisabled()

    const discardBtn = container.querySelector('button[aria-label=discard]')
    expect(discardBtn).not.toBeDisabled()

    await userEvent.click(saveBtn!)
    await waitFor(() => {
      expect(modifyRepository).toHaveBeenCalledWith({
        body: {
          ...MockGetHelmRegistryResponseWithAllData.content.data,
          description: 'updated description'
        },
        registry_ref: 'undefined/abcd/+'
      })
    })
  })

  test('should able to discard the changes', async () => {
    const { container } = render(
      <ArTestWrapper path="/registries/abcd/:tab" pathParams={{ tab: 'configuration' }}>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )

    const descriptionField = queryByNameAttribute('description', container)
    fireEvent.change(descriptionField!, { target: { value: 'updated description' } })
    expect(descriptionField).toHaveTextContent('updated description')

    const saveBtn = container.querySelector('button[aria-label=save]')
    expect(saveBtn).not.toBeDisabled()

    const discardBtn = container.querySelector('button[aria-label=discard]')
    expect(discardBtn).not.toBeDisabled()

    await userEvent.click(discardBtn!)
    await waitFor(() => {
      expect(saveBtn).toBeDisabled()
      expect(discardBtn).toBeDisabled()
      expect(descriptionField).toHaveTextContent(MockGetHelmRegistryResponseWithAllData.content.data.description)
    })
  })

  test('should render retry if failed to load get registry api', async () => {
    const refetchFn = jest.fn()
    ;(useGetRegistryQuery as jest.Mock).mockImplementation(() => ({
      isFetching: false,
      isError: true,
      error: { message: 'failed to load registry' },
      data: null,
      refetch: refetchFn
    }))

    const { container } = render(
      <ArTestWrapper queryParams={{ tab: 'configuration' }}>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )

    expect(getByText(container, 'failed to load registry')).toBeInTheDocument()
    const retryBtn = container.querySelector('button[aria-label=Retry]')
    expect(retryBtn).toBeInTheDocument()
    await userEvent.click(retryBtn!)
    await waitFor(() => {
      expect(refetchFn).toHaveBeenCalled()
    })
  })
})
