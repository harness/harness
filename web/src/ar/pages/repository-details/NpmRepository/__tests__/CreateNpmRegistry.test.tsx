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
import { fireEvent, getByTestId, getByText, render, screen, waitFor } from '@testing-library/react'

import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import RepositoryListPage from '@ar/pages/repository-list/RepositoryListPage'

import { queryByNameAttribute } from 'utils/test/testUtils'
import { MockGetNpmRegistryResponseWithAllData } from './__mockData__'
import '../../RepositoryFactory'

const createRegistryFn = jest.fn().mockImplementation(() => Promise.resolve(MockGetNpmRegistryResponseWithAllData))
const showSuccessToast = jest.fn()
const showErrorToast = jest.fn()
const mockHistoryPush = jest.fn()

jest.mock('@harnessio/uicore', () => ({
  ...jest.requireActual('@harnessio/uicore'),
  useToaster: jest.fn().mockImplementation(() => ({
    showSuccess: showSuccessToast,
    showError: showErrorToast,
    clear: jest.fn()
  }))
}))

// eslint-disable-next-line jest-no-mock
jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useHistory: () => ({
    push: mockHistoryPush
  })
}))

jest.mock('@harnessio/react-har-service-client', () => ({
  useGetAllRegistriesQuery: jest.fn().mockImplementation(() => ({
    isFetching: false,
    data: { content: { data: { registries: [] }, status: 'SUCCESS' } },
    refetch: jest.fn(),
    error: null
  })),
  useCreateRegistryMutation: jest.fn().mockImplementation(() => ({
    mutateAsync: createRegistryFn
  }))
}))

describe('Verify create npm registry flow', () => {
  test('Verify Modal header', async () => {
    const { container } = render(
      <ArTestWrapper
        featureFlags={{
          HAR_NPM_PACKAGE_TYPE_ENABLED: true
        }}>
        <RepositoryListPage />
      </ArTestWrapper>
    )

    const pageSubHeader = getByTestId(container, 'page-subheader')
    const createRegistryButton = getByText(pageSubHeader, 'repositoryList.newRepository')
    expect(createRegistryButton).toBeInTheDocument()
    await userEvent.click(createRegistryButton)

    const modal = document.getElementsByClassName('bp3-dialog')[0]
    expect(modal).toBeInTheDocument()

    const dialogHeader = screen.getByTestId('modaldialog-header')
    expect(dialogHeader).toHaveTextContent('repositoryDetails.repositoryForm.modalTitle')
    expect(dialogHeader).toHaveTextContent('repositoryDetails.repositoryForm.modalSubTitle')

    const closeButton = modal.querySelector('button[aria-label="Close"]')
    expect(closeButton).toBeInTheDocument()
    await userEvent.click(closeButton!)

    expect(modal).not.toBeInTheDocument()
  })

  test('verify registry type selector', async () => {
    const { container } = render(
      <ArTestWrapper
        featureFlags={{
          HAR_NPM_PACKAGE_TYPE_ENABLED: true
        }}>
        <RepositoryListPage />
      </ArTestWrapper>
    )

    const pageSubHeader = getByTestId(container, 'page-subheader')
    const createRegistryButton = getByText(pageSubHeader, 'repositoryList.newRepository')
    expect(createRegistryButton).toBeInTheDocument()
    await userEvent.click(createRegistryButton)

    const modal = document.getElementsByClassName('bp3-dialog')[0]
    expect(modal).toBeInTheDocument()

    const dialogBody = screen.getByTestId('modaldialog-body')
    expect(dialogBody).toBeInTheDocument()

    expect(dialogBody).toHaveTextContent('repositoryDetails.repositoryForm.selectRepoType')
    const registryTypeOption = dialogBody.querySelector('input[type="checkbox"][name=packageType][value="NPM"]')
    expect(registryTypeOption).not.toBeDisabled()
    fireEvent.change(registryTypeOption!, { target: { checked: true } })
    expect(registryTypeOption).toBeChecked()
  })

  test('verify npm registry create form with success scenario', async () => {
    const { container } = render(
      <ArTestWrapper
        featureFlags={{
          HAR_NPM_PACKAGE_TYPE_ENABLED: true
        }}>
        <RepositoryListPage />
      </ArTestWrapper>
    )

    const pageSubHeader = getByTestId(container, 'page-subheader')
    const createRegistryButton = getByText(pageSubHeader, 'repositoryList.newRepository')
    expect(createRegistryButton).toBeInTheDocument()
    await userEvent.click(createRegistryButton)

    const dialogBody = screen.getByTestId('modaldialog-body')
    expect(dialogBody).toBeInTheDocument()

    expect(dialogBody).toHaveTextContent('repositoryDetails.repositoryForm.selectRepoType')
    const registryTypeOption = dialogBody.querySelector('input[type="checkbox"][name=packageType][value="NPM"]')
    expect(registryTypeOption).not.toBeDisabled()
    await userEvent.click(registryTypeOption!)
    fireEvent.change(registryTypeOption!, { target: { checked: true } })
    expect(registryTypeOption).toBeChecked()

    expect(dialogBody).toHaveTextContent('repositoryDetails.repositoryForm.title')

    const formData = MockGetNpmRegistryResponseWithAllData.content.data
    const nameField = queryByNameAttribute('identifier', dialogBody)
    expect(nameField).toBeInTheDocument()
    expect(nameField).not.toBeDisabled()
    fireEvent.change(nameField!, { target: { value: formData.identifier } })

    const descriptionEditButton = getByTestId(dialogBody, 'description-edit')
    expect(descriptionEditButton).toBeInTheDocument()
    await userEvent.click(descriptionEditButton)
    const descriptionField = queryByNameAttribute('description', dialogBody)
    expect(descriptionField).toBeInTheDocument()
    expect(descriptionField).not.toBeDisabled()
    fireEvent.change(descriptionField!, { target: { value: formData.description } })

    const dialogFooter = screen.getByTestId('modaldialog-footer')
    expect(dialogFooter).toBeInTheDocument()
    const createButton = dialogFooter.querySelector(
      'button[type="submit"][aria-label="repositoryDetails.repositoryForm.create"]'
    )
    expect(createButton).toBeInTheDocument()
    await userEvent.click(createButton!)

    await waitFor(() => {
      expect(createRegistryFn).toHaveBeenCalledWith({
        body: {
          cleanupPolicy: [],
          config: { type: 'VIRTUAL', upstreamProxies: [] },
          description: 'custom description',
          identifier: 'npm-repo',
          packageType: 'NPM',
          parentRef: 'undefined',
          scanners: []
        },
        queryParams: { space_ref: 'undefined' }
      })
      expect(showSuccessToast).toHaveBeenCalledWith('repositoryDetails.repositoryForm.repositoryCreated')
      expect(mockHistoryPush).toHaveBeenCalledWith('/registries/npm-repo/configuration')
    })
  })

  test('verify npm registry create form with failure scenario', async () => {
    createRegistryFn.mockImplementation(() => Promise.reject({ message: 'error message' }))
    const { container } = render(
      <ArTestWrapper
        featureFlags={{
          HAR_NPM_PACKAGE_TYPE_ENABLED: true
        }}>
        <RepositoryListPage />
      </ArTestWrapper>
    )

    const pageSubHeader = getByTestId(container, 'page-subheader')
    const createRegistryButton = getByText(pageSubHeader, 'repositoryList.newRepository')
    expect(createRegistryButton).toBeInTheDocument()
    await userEvent.click(createRegistryButton)

    const dialogBody = screen.getByTestId('modaldialog-body')
    expect(dialogBody).toBeInTheDocument()

    expect(dialogBody).toHaveTextContent('repositoryDetails.repositoryForm.selectRepoType')
    const registryTypeOption = dialogBody.querySelector('input[type="checkbox"][name=packageType][value="NPM"]')
    expect(registryTypeOption).not.toBeDisabled()
    await userEvent.click(registryTypeOption!)
    fireEvent.change(registryTypeOption!, { target: { checked: true } })
    expect(registryTypeOption).toBeChecked()

    expect(dialogBody).toHaveTextContent('repositoryDetails.repositoryForm.title')

    const formData = MockGetNpmRegistryResponseWithAllData.content.data
    const nameField = queryByNameAttribute('identifier', dialogBody)
    expect(nameField).toBeInTheDocument()
    expect(nameField).not.toBeDisabled()
    fireEvent.change(nameField!, { target: { value: formData.identifier } })

    const dialogFooter = screen.getByTestId('modaldialog-footer')
    expect(dialogFooter).toBeInTheDocument()
    const createButton = dialogFooter.querySelector(
      'button[type="submit"][aria-label="repositoryDetails.repositoryForm.create"]'
    )
    expect(createButton).toBeInTheDocument()
    await userEvent.click(createButton!)

    await waitFor(() => {
      expect(createRegistryFn).toHaveBeenCalledWith({
        body: {
          cleanupPolicy: [],
          config: { type: 'VIRTUAL', upstreamProxies: [] },
          identifier: 'npm-repo',
          packageType: 'NPM',
          parentRef: 'undefined',
          scanners: []
        },
        queryParams: { space_ref: 'undefined' }
      })
      expect(showErrorToast).toHaveBeenCalledWith('error message')
    })
  })
})
