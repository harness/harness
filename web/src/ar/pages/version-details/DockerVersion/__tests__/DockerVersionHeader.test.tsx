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
import { getByTestId, render, waitFor } from '@testing-library/react'
import {
  type DockerManifestDetails,
  getAllArtifactVersions,
  useGetDockerArtifactManifestsQuery
} from '@harnessio/react-har-service-client'

import '@ar/pages/version-details/VersionFactory'
import '@ar/pages/repository-details/RepositoryFactory'
import { testSelectChange } from '@ar/utils/testUtils/utils'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'

import VersionDetailsPage from '../../VersionDetailsPage'
import { mockDockerManifestList, mockDockerVersionList, mockDockerVersionSummary } from './__mockData__'

const mockHistoryPush = jest.fn()
// eslint-disable-next-line jest-no-mock
jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useHistory: () => ({
    push: mockHistoryPush
  })
}))

jest.mock('@harnessio/react-har-service-client', () => ({
  useGetArtifactVersionSummaryQuery: jest.fn().mockImplementation(() => ({
    data: { content: mockDockerVersionSummary },
    error: null,
    isLoading: false,
    refetch: jest.fn()
  })),
  useGetDockerArtifactManifestsQuery: jest.fn().mockImplementation(() => ({
    data: { content: mockDockerManifestList },
    error: null,
    isLoading: false,
    refetch: jest.fn()
  })),
  getAllArtifactVersions: jest.fn().mockImplementation(
    () =>
      new Promise(success => {
        success({ content: mockDockerVersionList })
      })
  )
}))

describe('Verify DockerVersionHeader component render', () => {
  beforeAll(() => {
    jest.clearAllMocks()
  })
  test('Verify breadcrumbs', () => {
    const { container } = render(
      <ArTestWrapper>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    const pageHeader = getByTestId(container, 'page-header')
    expect(pageHeader).toBeInTheDocument()

    const breadcrumbsSection = pageHeader?.querySelector('div[class*=PageHeader--breadcrumbsDiv--]')
    expect(breadcrumbsSection).toBeInTheDocument()

    expect(breadcrumbsSection).toHaveTextContent('breadcrumbs.repositories: undefined')
    expect(breadcrumbsSection).toHaveTextContent('breadcrumbs.artifacts: undefined')
  })

  test('Verify icon, name, selectors and other actions', () => {
    const { container } = render(
      <ArTestWrapper>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    const pageHeader = getByTestId(container, 'page-header')
    expect(pageHeader).toBeInTheDocument()

    expect(pageHeader?.querySelector('span[data-icon=docker-step]')).toBeInTheDocument()

    const data = mockDockerVersionSummary.data
    expect(pageHeader).toHaveTextContent(data.imageName)

    const versionSelector = getByTestId(container, 'version-select')
    expect(versionSelector).toBeInTheDocument()

    const archSelector = getByTestId(container, 'version-arch-select')
    expect(archSelector).toBeInTheDocument()

    const setupClientBtn = pageHeader.querySelector('button[aria-label="actions.setupClient"]')
    expect(setupClientBtn).toBeInTheDocument()
  })

  test('verify version selector: Success Case', async () => {
    const { container } = render(
      <ArTestWrapper>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    const versionSelector = getByTestId(container, 'version-select')
    expect(versionSelector).toBeInTheDocument()

    const data = mockDockerVersionSummary.data
    expect(versionSelector).toHaveTextContent(data.version)

    await userEvent.click(versionSelector)
    await testSelectChange(versionSelector, '1.0.1', data.version)

    await waitFor(() => {
      expect(mockHistoryPush).toHaveBeenLastCalledWith('/registries/artifacts/versions/1.0.1')
    })
  })

  test('verify version selector: Failure Case', async () => {
    ;(getAllArtifactVersions as jest.Mock).mockImplementationOnce(
      () =>
        new Promise((_, failure) => {
          failure({ message: 'Failed to fetch versions' })
        })
    )
    const { container } = render(
      <ArTestWrapper>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    const versionSelector = getByTestId(container, 'version-select')
    expect(versionSelector).toBeInTheDocument()

    const data = mockDockerVersionSummary.data
    expect(versionSelector).toHaveTextContent(data.version)

    await userEvent.click(versionSelector)
    const dialogs = document.getElementsByClassName('bp3-select-popover')
    await waitFor(() => expect(dialogs).toHaveLength(1))
    const selectPopover = dialogs[0] as HTMLElement
    expect(selectPopover).toHaveTextContent('No items found')
  })

  test('verify arch selector: Success Case', async () => {
    const { container } = render(
      <ArTestWrapper
        queryParams={{
          digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    const archSelector = getByTestId(container, 'version-arch-select')
    expect(archSelector).toBeInTheDocument()

    const data = mockDockerManifestList.data.manifests?.[0] as DockerManifestDetails
    await waitFor(() => {
      expect(archSelector).toHaveTextContent(data.osArch)
    })

    await userEvent.click(archSelector)
    await testSelectChange(archSelector, 'linux/amd64', 'linux/arm64')

    await waitFor(() => {
      expect(mockHistoryPush).toHaveBeenLastCalledWith(
        '/?digest=sha256%3A112cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
      )
    })
  })

  test('verify arch selector: Loading Case', async () => {
    ;(useGetDockerArtifactManifestsQuery as jest.Mock).mockImplementation(() => ({
      data: null,
      error: null,
      isFetching: true,
      refetch: jest.fn()
    }))

    const { container } = render(
      <ArTestWrapper
        queryParams={{
          digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    const archSelector = getByTestId(container, 'version-arch-select')
    expect(archSelector).toBeInTheDocument()

    await userEvent.click(archSelector)
    const dialogs = document.getElementsByClassName('bp3-select-popover')
    await waitFor(() => expect(dialogs).toHaveLength(1))
    const selectPopover = dialogs[0] as HTMLElement
    expect(selectPopover).toHaveTextContent('Loading')
  })

  test('verify arch selector: Failure Case', async () => {
    ;(useGetDockerArtifactManifestsQuery as jest.Mock).mockImplementation(() => ({
      data: null,
      error: { message: 'Failed to fetch manifests' },
      isFetching: false,
      refetch: jest.fn()
    }))

    const { container } = render(
      <ArTestWrapper
        queryParams={{
          digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    const archSelector = getByTestId(container, 'version-arch-select')
    expect(archSelector).toBeInTheDocument()

    await userEvent.click(archSelector)
    const dialogs = document.getElementsByClassName('bp3-select-popover')
    await waitFor(() => expect(dialogs).toHaveLength(1))
    const selectPopover = dialogs[0] as HTMLElement
    expect(selectPopover).toHaveTextContent('Failed to fetch manifests')
  })

  test('verify arch selector: Empty response Case', async () => {
    ;(useGetDockerArtifactManifestsQuery as jest.Mock).mockImplementation(() => ({
      data: { content: { data: { manifests: null } } },
      error: null,
      isFetching: false,
      refetch: jest.fn()
    }))

    const { container } = render(
      <ArTestWrapper
        queryParams={{
          digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    const archSelector = getByTestId(container, 'version-arch-select')
    expect(archSelector).toBeInTheDocument()

    await userEvent.click(archSelector)
    const dialogs = document.getElementsByClassName('bp3-select-popover')
    await waitFor(() => expect(dialogs).toHaveLength(1))
    const selectPopover = dialogs[0] as HTMLElement
    expect(selectPopover).toHaveTextContent('No items found')
  })
})
