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
import { render, screen, waitFor } from '@testing-library/react'
import type { Registry } from '@harnessio/react-har-service-client'

import { Parent } from '@ar/common/types'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import RepositoryListPage from '@ar/pages/repository-list/RepositoryListPage'

import { MockGetDockerUpstreamRegistryResponseWithDockerhubSourceAllData } from './__mockData__'
import upstreamProxyUtils from '../../__tests__/utils'
import '../../RepositoryFactory'

const createRegistryFn = jest
  .fn()
  .mockImplementation(() => Promise.resolve(MockGetDockerUpstreamRegistryResponseWithDockerhubSourceAllData))
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

describe('Verify create docker upstream registry flow', () => {
  test('Verify Modal header', async () => {
    const { container } = render(
      <ArTestWrapper>
        <RepositoryListPage />
      </ArTestWrapper>
    )

    const modal = await upstreamProxyUtils.openModal(container)

    const dialogHeader = screen.getByTestId('modaldialog-header')
    expect(dialogHeader).toHaveTextContent('upstreamProxyDetails.createForm.title')

    const closeButton = modal.querySelector('button[aria-label="Close"]')
    expect(closeButton).toBeInTheDocument()
    await userEvent.click(closeButton!)

    expect(modal).not.toBeInTheDocument()
  })

  test('verify registry type selector', async () => {
    const { container } = render(
      <ArTestWrapper>
        <RepositoryListPage />
      </ArTestWrapper>
    )

    await upstreamProxyUtils.openModal(container)

    const dialogBody = screen.getByTestId('modaldialog-body')
    expect(dialogBody).toBeInTheDocument()

    await upstreamProxyUtils.verifyPackageTypeSelector(dialogBody, 'DOCKER')
  })

  test('verify docker registry create form with success scenario > Source as DockerHub > Anonymous', async () => {
    const { container } = render(
      <ArTestWrapper>
        <RepositoryListPage />
      </ArTestWrapper>
    )

    await upstreamProxyUtils.openModal(container)

    const dialogBody = screen.getByTestId('modaldialog-body')
    expect(dialogBody).toBeInTheDocument()

    await upstreamProxyUtils.verifyPackageTypeSelector(dialogBody, 'DOCKER')

    expect(dialogBody).toHaveTextContent('upstreamProxyDetails.form.title')

    const formData = MockGetDockerUpstreamRegistryResponseWithDockerhubSourceAllData.content.data as Registry

    await upstreamProxyUtils.verifyUpstreamProxyCreateForm(dialogBody, formData, 'Dockerhub', 'Anonymous')

    const createButton = await upstreamProxyUtils.getSubmitButton()
    await userEvent.click(createButton!)

    await waitFor(() => {
      expect(createRegistryFn).toHaveBeenLastCalledWith({
        body: {
          cleanupPolicy: [],
          config: { auth: null, authType: 'Anonymous', source: 'Dockerhub', type: 'UPSTREAM', url: '' },
          description: 'test description',
          identifier: 'docker-up-repo',
          packageType: 'DOCKER',
          parentRef: 'undefined',
          scanners: []
        },
        queryParams: { space_ref: 'undefined' }
      })
      expect(showSuccessToast).toHaveBeenCalledWith(
        'upstreamProxyDetails.actions.createUpdateModal.createSuccessMessage'
      )
      expect(mockHistoryPush).toHaveBeenCalledWith('/registries/docker-up-repo/configuration')
    })
  })

  test('verify docker registry create form with success scenario > Source as DockerHub > Username Password', async () => {
    const { container } = render(
      <ArTestWrapper parent={Parent.OSS}>
        <RepositoryListPage />
      </ArTestWrapper>
    )

    await upstreamProxyUtils.openModal(container)

    const dialogBody = screen.getByTestId('modaldialog-body')
    expect(dialogBody).toBeInTheDocument()

    await upstreamProxyUtils.verifyPackageTypeSelector(dialogBody, 'DOCKER')

    expect(dialogBody).toHaveTextContent('upstreamProxyDetails.form.title')

    const formData = MockGetDockerUpstreamRegistryResponseWithDockerhubSourceAllData.content.data as Registry

    await upstreamProxyUtils.verifyUpstreamProxyCreateForm(dialogBody, formData, 'Dockerhub', 'UserPassword')

    const createButton = await upstreamProxyUtils.getSubmitButton()
    await userEvent.click(createButton!)

    await waitFor(() => {
      expect(createRegistryFn).toHaveBeenLastCalledWith({
        body: {
          cleanupPolicy: [],
          config: {
            auth: { authType: 'UserPassword', secretIdentifier: 'password', userName: 'username' },
            authType: 'UserPassword',
            source: 'Dockerhub',
            type: 'UPSTREAM',
            url: ''
          },
          description: 'test description',
          identifier: 'docker-up-repo',
          packageType: 'DOCKER',
          parentRef: 'undefined',
          scanners: []
        },
        queryParams: { space_ref: 'undefined' }
      })
      expect(showSuccessToast).toHaveBeenCalledWith(
        'upstreamProxyDetails.actions.createUpdateModal.createSuccessMessage'
      )
      expect(mockHistoryPush).toHaveBeenCalledWith('/registries/docker-up-repo/configuration')
    })
  })

  test('verify docker registry create form with success scenario > Source as AWS ECR > Anonymous', async () => {
    const { container } = render(
      <ArTestWrapper>
        <RepositoryListPage />
      </ArTestWrapper>
    )

    await upstreamProxyUtils.openModal(container)

    const dialogBody = screen.getByTestId('modaldialog-body')
    expect(dialogBody).toBeInTheDocument()

    await upstreamProxyUtils.verifyPackageTypeSelector(dialogBody, 'DOCKER')

    expect(dialogBody).toHaveTextContent('upstreamProxyDetails.form.title')

    const formData = MockGetDockerUpstreamRegistryResponseWithDockerhubSourceAllData.content.data as Registry

    await upstreamProxyUtils.verifyUpstreamProxyCreateForm(dialogBody, formData, 'AwsEcr', 'Anonymous')

    const createButton = await upstreamProxyUtils.getSubmitButton()
    await userEvent.click(createButton!)

    await waitFor(() => {
      expect(createRegistryFn).toHaveBeenLastCalledWith({
        body: {
          cleanupPolicy: [],
          config: { auth: null, authType: 'Anonymous', source: 'AwsEcr', type: 'UPSTREAM', url: 'https://aws.ecr.com' },
          description: 'test description',
          identifier: 'docker-up-repo',
          packageType: 'DOCKER',
          parentRef: 'undefined',
          scanners: []
        },
        queryParams: { space_ref: 'undefined' }
      })
      expect(showSuccessToast).toHaveBeenCalledWith(
        'upstreamProxyDetails.actions.createUpdateModal.createSuccessMessage'
      )
      expect(mockHistoryPush).toHaveBeenCalledWith('/registries/docker-up-repo/configuration')
    })
  })

  test('verify docker registry create form with success scenario > Source as AWS ECR > Access Key and Secret Key', async () => {
    const { container } = render(
      <ArTestWrapper parent={Parent.OSS}>
        <RepositoryListPage />
      </ArTestWrapper>
    )

    await upstreamProxyUtils.openModal(container)

    const dialogBody = screen.getByTestId('modaldialog-body')
    expect(dialogBody).toBeInTheDocument()

    await upstreamProxyUtils.verifyPackageTypeSelector(dialogBody, 'DOCKER')

    expect(dialogBody).toHaveTextContent('upstreamProxyDetails.form.title')

    const formData = MockGetDockerUpstreamRegistryResponseWithDockerhubSourceAllData.content.data as Registry

    await upstreamProxyUtils.verifyUpstreamProxyCreateForm(dialogBody, formData, 'AwsEcr', 'AccessKeySecretKey')

    const createButton = await upstreamProxyUtils.getSubmitButton()
    await userEvent.click(createButton!)

    await waitFor(() => {
      expect(createRegistryFn).toHaveBeenLastCalledWith({
        body: {
          cleanupPolicy: [],
          config: {
            auth: {
              accessKey: 'accessKey',
              accessKeyType: 'TEXT',
              secretKeyIdentifier: 'secretKey',
              authType: 'AccessKeySecretKey'
            },
            authType: 'AccessKeySecretKey',
            source: 'AwsEcr',
            type: 'UPSTREAM',
            url: 'https://aws.ecr.com'
          },
          description: 'test description',
          identifier: 'docker-up-repo',
          packageType: 'DOCKER',
          parentRef: 'undefined',
          scanners: []
        },
        queryParams: { space_ref: 'undefined' }
      })
      expect(showSuccessToast).toHaveBeenCalledWith(
        'upstreamProxyDetails.actions.createUpdateModal.createSuccessMessage'
      )
      expect(mockHistoryPush).toHaveBeenCalledWith('/registries/docker-up-repo/configuration')
    })
  })

  test('verify docker registry create form with success scenario > Source as Custom > Anonymous', async () => {
    const { container } = render(
      <ArTestWrapper>
        <RepositoryListPage />
      </ArTestWrapper>
    )

    await upstreamProxyUtils.openModal(container)

    const dialogBody = screen.getByTestId('modaldialog-body')
    expect(dialogBody).toBeInTheDocument()

    await upstreamProxyUtils.verifyPackageTypeSelector(dialogBody, 'DOCKER')

    expect(dialogBody).toHaveTextContent('upstreamProxyDetails.form.title')

    const formData = MockGetDockerUpstreamRegistryResponseWithDockerhubSourceAllData.content.data as Registry

    await upstreamProxyUtils.verifyUpstreamProxyCreateForm(dialogBody, formData, 'Custom', 'Anonymous')

    const createButton = await upstreamProxyUtils.getSubmitButton()
    await userEvent.click(createButton!)

    await waitFor(() => {
      expect(createRegistryFn).toHaveBeenLastCalledWith({
        body: {
          cleanupPolicy: [],
          config: {
            auth: null,
            authType: 'Anonymous',
            source: 'Custom',
            type: 'UPSTREAM',
            url: 'https://custom.docker.com'
          },
          description: 'test description',
          identifier: 'docker-up-repo',
          packageType: 'DOCKER',
          parentRef: 'undefined',
          scanners: []
        },
        queryParams: { space_ref: 'undefined' }
      })
      expect(showSuccessToast).toHaveBeenCalledWith(
        'upstreamProxyDetails.actions.createUpdateModal.createSuccessMessage'
      )
      expect(mockHistoryPush).toHaveBeenCalledWith('/registries/docker-up-repo/configuration')
    })
  })

  test('verify docker registry create form with success scenario > Source as Custom > Username Password', async () => {
    const { container } = render(
      <ArTestWrapper parent={Parent.OSS}>
        <RepositoryListPage />
      </ArTestWrapper>
    )

    await upstreamProxyUtils.openModal(container)

    const dialogBody = screen.getByTestId('modaldialog-body')
    expect(dialogBody).toBeInTheDocument()

    await upstreamProxyUtils.verifyPackageTypeSelector(dialogBody, 'DOCKER')

    expect(dialogBody).toHaveTextContent('upstreamProxyDetails.form.title')

    const formData = MockGetDockerUpstreamRegistryResponseWithDockerhubSourceAllData.content.data as Registry

    await upstreamProxyUtils.verifyUpstreamProxyCreateForm(dialogBody, formData, 'Custom', 'UserPassword')

    const createButton = await upstreamProxyUtils.getSubmitButton()
    await userEvent.click(createButton!)

    await waitFor(() => {
      expect(createRegistryFn).toHaveBeenLastCalledWith({
        body: {
          cleanupPolicy: [],
          config: {
            auth: { authType: 'UserPassword', secretIdentifier: 'password', userName: 'username' },
            authType: 'UserPassword',
            source: 'Custom',
            type: 'UPSTREAM',
            url: 'https://custom.docker.com'
          },
          description: 'test description',
          identifier: 'docker-up-repo',
          packageType: 'DOCKER',
          parentRef: 'undefined',
          scanners: []
        },
        queryParams: { space_ref: 'undefined' }
      })
      expect(showSuccessToast).toHaveBeenCalledWith(
        'upstreamProxyDetails.actions.createUpdateModal.createSuccessMessage'
      )
      expect(mockHistoryPush).toHaveBeenCalledWith('/registries/docker-up-repo/configuration')
    })
  })
})
