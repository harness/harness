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

import { Parent } from '@ar/common/types'
import { DEFAULT_DATE_TIME_FORMAT } from '@ar/constants'
import { getReadableDateTime } from '@ar/common/dateUtils'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'

import { queryByNameAttribute } from 'utils/test/testUtils'
import RepositoryDetailsPage from '../../RepositoryDetailsPage'
import {
  MockGetHelmArtifactsByRegistryResponse,
  MockGetHelmUpstreamRegistryResponseWithCustomSourceAllData
} from './__mockData__'
import upstreamProxyUtils from '../../__tests__/utils'

const modifyRepository = jest.fn().mockImplementation(
  () =>
    new Promise(onSuccess => {
      onSuccess({ content: { status: 'SUCCESS' } })
    })
)

const showSuccessToast = jest.fn()
const showErrorToast = jest.fn()

jest.mock('@harnessio/uicore', () => ({
  ...jest.requireActual('@harnessio/uicore'),
  useToaster: jest.fn().mockImplementation(() => ({
    showSuccess: showSuccessToast,
    showError: showErrorToast,
    clear: jest.fn()
  }))
}))

const mockHistoryPush = jest.fn()
// eslint-disable-next-line jest-no-mock
jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useHistory: () => ({
    push: mockHistoryPush
  })
}))

jest.mock('@harnessio/react-har-service-client', () => ({
  useListPackagesQuery: jest.fn(),
  useGetRegistryQuery: jest.fn().mockImplementation(() => ({
    isFetching: false,
    refetch: jest.fn(),
    error: false,
    data: MockGetHelmUpstreamRegistryResponseWithCustomSourceAllData
  })),
  useGetAllArtifactsByRegistryQuery: jest.fn().mockImplementation(() => ({
    isFetching: false,
    refetch: jest.fn(),
    error: false,
    data: MockGetHelmArtifactsByRegistryResponse
  })),
  useDeleteRegistryMutation: jest.fn().mockImplementation(() => ({
    isLoading: false,
    mutateAsync: jest.fn()
  })),
  useModifyRegistryMutation: jest.fn().mockImplementation(() => ({
    isLoading: false,
    mutateAsync: modifyRepository
  }))
}))

describe('Verify header section for docker artifact registry', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  test('Verify breadcrumbs', async () => {
    const { container } = render(
      <ArTestWrapper parent={Parent.OSS}>
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
    const pageHeader = getByTestId(container, 'upstream-registry-header-container')
    expect(pageHeader).toBeInTheDocument()

    expect(pageHeader?.querySelector('span[data-icon=service-helm]')).toBeInTheDocument()
    const data = MockGetHelmUpstreamRegistryResponseWithCustomSourceAllData.content.data

    expect(pageHeader).toHaveTextContent(data.identifier)

    expect(pageHeader).toHaveTextContent(getReadableDateTime(Number(data.modifiedAt), DEFAULT_DATE_TIME_FORMAT))
  })

  test('Verify registry setup client action not visible', async () => {
    const { container } = render(
      <ArTestWrapper>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )
    const pageHeader = getByTestId(container, 'upstream-registry-header-container')
    const setupClientBtn = pageHeader.querySelector('button[aria-label="actions.setupClient"]')
    expect(setupClientBtn).toBeInTheDocument()
  })

  test('Verify other registry actions', async () => {
    const { container } = render(
      <ArTestWrapper>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )
    const pageHeader = getByTestId(container, 'upstream-registry-header-container')
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
    const registryDefinitionSection = getByTestId(container, 'upstream-registry-definition')
    const nameField = queryByNameAttribute('identifier', registryDefinitionSection)
    expect(nameField).toBeInTheDocument()
    expect(nameField).toBeDisabled()
    expect(nameField).toHaveAttribute(
      'value',
      MockGetHelmUpstreamRegistryResponseWithCustomSourceAllData.content.data.identifier
    )

    const descriptionField = queryByNameAttribute('description', registryDefinitionSection)
    expect(descriptionField).toBeInTheDocument()
    expect(descriptionField).not.toBeDisabled()
    expect(descriptionField).toHaveTextContent(
      MockGetHelmUpstreamRegistryResponseWithCustomSourceAllData.content.data.description
    )

    const tags = registryDefinitionSection.querySelectorAll('div.bp3-tag-input-values .bp3-tag')
    tags.forEach((each, idx) => {
      expect(each).toHaveTextContent(
        MockGetHelmUpstreamRegistryResponseWithCustomSourceAllData.content.data.labels[idx]
      )
    })

    // verify source selection
    const sourceAuthSection = getByTestId(container, 'upstream-source-auth-definition')
    const sourceSection = sourceAuthSection.querySelector('input[type=radio][name="config.source"][value=Custom]')
    expect(sourceSection).toBeChecked()
    expect(sourceSection).not.toBeDisabled()

    // verify auth type selection
    const authTypeSelection = sourceAuthSection.querySelector(
      'input[type=radio][name="config.authType"][value=Anonymous]'
    )
    expect(authTypeSelection).toBeChecked()
    expect(authTypeSelection).not.toBeDisabled()

    // Security scan section
    const securityScanSection = queryByTestId(container, 'security-section')
    expect(securityScanSection).toBeInTheDocument()

    // artifact filtering rules
    const filteringRulesSection = getByTestId(container, 'include-exclude-patterns-section')
    expect(filteringRulesSection).toBeInTheDocument()

    const allowedPatternsSection = filteringRulesSection.querySelectorAll('div.bp3-form-group')[0]
    const allowedPatterns = allowedPatternsSection.querySelectorAll('div.bp3-tag-input-values .bp3-tag')
    allowedPatterns.forEach((each, idx) => {
      expect(each).toHaveTextContent(
        MockGetHelmUpstreamRegistryResponseWithCustomSourceAllData.content.data.allowedPattern[idx]
      )
    })

    const advancedOptionTitle = getByText(container, 'repositoryDetails.repositoryForm.advancedOptionsTitle')
    expect(advancedOptionTitle).toBeInTheDocument()
    await userEvent.click(advancedOptionTitle)

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

  test('should able to submit the form with updated data: Success Scenario', async () => {
    const { container } = render(
      <ArTestWrapper path="/registries/abcd/:tab" pathParams={{ tab: 'configuration' }}>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )

    const descriptionField = queryByNameAttribute('description', container)
    fireEvent.change(descriptionField!, { target: { value: 'updated description' } })
    expect(descriptionField).toHaveTextContent('updated description')

    const sourceAuthSection = getByTestId(container, 'upstream-source-auth-definition')
    let sourceSection = sourceAuthSection.querySelector('input[type=radio][name="config.authType"][value=UserPassword]')
    await userEvent.click(sourceSection!)
    sourceSection = sourceAuthSection.querySelector('input[type=radio][name="config.authType"][value=Anonymous]')
    await userEvent.click(sourceSection!)

    const saveBtn = container.querySelector('button[aria-label=save]')
    expect(saveBtn).not.toBeDisabled()

    const discardBtn = container.querySelector('button[aria-label=discard]')
    expect(discardBtn).not.toBeDisabled()

    await userEvent.click(saveBtn!)
    await waitFor(() => {
      expect(modifyRepository).toHaveBeenLastCalledWith({
        body: {
          ...MockGetHelmUpstreamRegistryResponseWithCustomSourceAllData.content.data,
          description: 'updated description',
          parentRef: 'undefined/+'
        },
        registry_ref: 'undefined/abcd/+'
      })
      expect(showSuccessToast).toHaveBeenLastCalledWith(
        'upstreamProxyDetails.actions.createUpdateModal.updateSuccessMessage'
      )
    })
  })

  test('Verify source and auth section with multiple scenarios', async () => {
    const { container } = render(
      <ArTestWrapper parent={Parent.OSS} path="/registries/abcd/:tab" pathParams={{ tab: 'configuration' }}>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )

    const saveBtn = container.querySelector('button[aria-label=save]')

    const sourceAuthSection = getByTestId(container, 'upstream-source-auth-definition')

    // verify Custom, UserPassword
    {
      await upstreamProxyUtils.verifySourceAndAuthSection(
        sourceAuthSection,
        'Custom',
        'UserPassword',
        'Custom',
        'Anonymous'
      )

      userEvent.click(saveBtn!)
      await waitFor(() => {
        expect(modifyRepository).toHaveBeenLastCalledWith({
          body: {
            allowedPattern: ['test1', 'test2'],
            config: {
              auth: { authType: 'UserPassword', secretIdentifier: 'password', userName: 'username' },
              authType: 'UserPassword',
              source: 'Custom',
              type: 'UPSTREAM',
              url: 'https://custom.docker.com'
            },
            createdAt: '1738516362995',
            description: 'test description',
            identifier: 'helm-up-repo',
            labels: ['label1', 'label2', 'label3', 'label4'],
            modifiedAt: '1738516362995',
            packageType: 'HELM',
            cleanupPolicy: [],
            url: '',
            isPublic: false,
            uuid: 'uuid',
            parentRef: 'undefined/+',
            isDeleted: false
          },
          registry_ref: 'undefined/abcd/+'
        })
      })
    }

    // verify AwsEcr, Anonymous
    {
      await upstreamProxyUtils.verifySourceAndAuthSection(
        sourceAuthSection,
        'AwsEcr',
        'Anonymous',
        'Custom',
        'Anonymous'
      )

      userEvent.click(saveBtn!)
      await waitFor(() => {
        expect(modifyRepository).toHaveBeenLastCalledWith({
          body: {
            allowedPattern: ['test1', 'test2'],
            config: {
              auth: null,
              authType: 'Anonymous',
              source: 'AwsEcr',
              type: 'UPSTREAM',
              url: 'https://aws.ecr.com',
              remoteUrlSuffix: ''
            },
            createdAt: '1738516362995',
            description: 'test description',
            identifier: 'helm-up-repo',
            labels: ['label1', 'label2', 'label3', 'label4'],
            modifiedAt: '1738516362995',
            packageType: 'HELM',
            cleanupPolicy: [],
            url: '',
            isPublic: false,
            uuid: 'uuid',
            parentRef: 'undefined/+',
            isDeleted: false
          },
          registry_ref: 'undefined/abcd/+'
        })
      })
    }

    // verify AwsEcr, AccessKeySecretKey
    {
      await upstreamProxyUtils.verifySourceAndAuthSection(
        sourceAuthSection,
        'AwsEcr',
        'AccessKeySecretKey',
        'Custom',
        'Anonymous'
      )

      userEvent.click(saveBtn!)
      await waitFor(() => {
        expect(modifyRepository).toHaveBeenLastCalledWith({
          body: {
            allowedPattern: ['test1', 'test2'],
            config: {
              auth: {
                accessKey: 'accessKey',
                accessKeyType: 'TEXT',
                authType: 'AccessKeySecretKey',
                secretKeyIdentifier: 'secretKey'
              },
              authType: 'AccessKeySecretKey',
              source: 'AwsEcr',
              type: 'UPSTREAM',
              url: 'https://aws.ecr.com',
              remoteUrlSuffix: ''
            },
            createdAt: '1738516362995',
            description: 'test description',
            identifier: 'helm-up-repo',
            labels: ['label1', 'label2', 'label3', 'label4'],
            modifiedAt: '1738516362995',
            packageType: 'HELM',
            cleanupPolicy: [],
            url: '',
            isPublic: false,
            uuid: 'uuid',
            parentRef: 'undefined/+',
            isDeleted: false
          },
          registry_ref: 'undefined/abcd/+'
        })
      })
    }
  })

  test('should able to submit the form with updated data: Failure Scenario', async () => {
    modifyRepository.mockImplementationOnce(
      () =>
        new Promise((_, onReject) => {
          onReject({ message: 'error message' })
        })
    )
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
      expect(modifyRepository).toHaveBeenLastCalledWith({
        body: {
          ...MockGetHelmUpstreamRegistryResponseWithCustomSourceAllData.content.data,
          description: 'updated description',
          parentRef: 'undefined/+'
        },
        registry_ref: 'undefined/abcd/+'
      })
      expect(showErrorToast).toHaveBeenLastCalledWith('error message')
    })
  })
})
