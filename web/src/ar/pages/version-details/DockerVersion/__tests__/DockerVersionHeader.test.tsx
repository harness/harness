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
  useGetArtifactVersionSummaryQuery,
  useGetDockerArtifactManifestsQuery
} from '@harnessio/react-har-service-client'

import '@ar/pages/version-details/VersionFactory'
import '@ar/pages/repository-details/RepositoryFactory'
import { testSelectChange } from '@ar/utils/testUtils/utils'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'

import VersionDetailsPage from '../../VersionDetailsPage'
import {
  mockDockerArtifactDetails,
  mockDockerArtifactIntegrationDetails,
  mockDockerArtifactLayers,
  mockDockerManifestList,
  mockDockerVersionList,
  mockDockerVersionSummary,
  mockDockerVersionSummaryWithoutSscaAndStoData
} from './__mockData__'

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
  ),
  useGetDockerArtifactDetailsQuery: jest.fn().mockImplementation(() => ({
    data: { content: mockDockerArtifactDetails },
    error: null,
    isLoading: false,
    refetch: jest.fn()
  })),
  useGetDockerArtifactIntegrationDetailsQuery: jest.fn().mockImplementation(() => ({
    data: { content: mockDockerArtifactIntegrationDetails },
    error: null,
    isLoading: false,
    refetch: jest.fn()
  })),
  useGetDockerArtifactLayersQuery: jest.fn().mockImplementation(() => ({
    data: { content: mockDockerArtifactLayers },
    error: null,
    isLoading: false,
    refetch: jest.fn()
  }))
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
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/:artifactType/:artifactIdentifier/versions"
        pathParams={{ repositoryIdentifier: 'reg1', artifactType: 'artifacts', artifactIdentifier: 'docker' }}>
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
      expect(mockHistoryPush).toHaveBeenCalledWith('/registries/reg1/artifacts/docker/versions/1.0.1')
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
      expect(mockHistoryPush).toHaveBeenCalledWith(
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

  test('verify tab navigation with ssca and sto data', async () => {
    const { container } = render(
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/:artifactType/:artifactIdentifier/versions/:versionIdentifier/overview"
        pathParams={{
          repositoryIdentifier: 'reg1',
          artifactType: 'artifacts',
          artifactIdentifier: 'docker',
          versionIdentifier: '1.0.1'
        }}
        queryParams={{
          digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    const overviewTab = container.querySelector('div[data-tab-id=overview]')
    await userEvent.click(overviewTab!)
    expect(mockHistoryPush).toHaveBeenLastCalledWith(
      '/registries/reg1/artifacts/docker/versions/1.0.1/overview?digest=sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
    )

    const artifactDetailsTab = container.querySelector('div[data-tab-id=artifact_details]')
    await userEvent.click(artifactDetailsTab!)
    expect(mockHistoryPush).toHaveBeenLastCalledWith(
      '/registries/reg1/artifacts/docker/versions/1.0.1/artifact_details?digest=sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
    )

    const sscaTab = container.querySelector('div[data-tab-id=supply_chain]')
    await userEvent.click(sscaTab!)
    expect(mockHistoryPush).toHaveBeenLastCalledWith(
      '/registries/reg1/artifacts/docker/versions/1.0.1/orgs/default/projects/default_project/artifact-sources/67a5dccf6d75916b0c3ea1b5/artifacts/67a5dccf6d75916b0c3ea1b6/supply_chain?digest=sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
    )

    const stoTab = container.querySelector('div[data-tab-id=security_tests]')
    await userEvent.click(stoTab!)
    expect(mockHistoryPush).toHaveBeenLastCalledWith(
      '/registries/reg1/artifacts/docker/versions/1.0.1/orgs/default/projects/default_project/pipelines/HARNESS_ARTIFACT_SCAN_PIPELINE/executions/Tbi7s6nETjmOMKU3Qrnm7A/security_tests?digest=sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
    )
  })

  test('verify tab navigation with ssca and sto data', async () => {
    ;(useGetArtifactVersionSummaryQuery as jest.Mock).mockImplementation(() => ({
      data: { content: mockDockerVersionSummaryWithoutSscaAndStoData },
      error: null,
      isLoading: false,
      refetch: jest.fn()
    }))
    const { container } = render(
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/:artifactType/:artifactIdentifier/versions/:versionIdentifier/overview"
        pathParams={{
          repositoryIdentifier: 'reg1',
          artifactType: 'artifacts',
          artifactIdentifier: 'docker',
          versionIdentifier: '1.0.1'
        }}
        queryParams={{
          digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    const overviewTab = container.querySelector('div[data-tab-id=overview]')
    await userEvent.click(overviewTab!)
    expect(mockHistoryPush).toHaveBeenLastCalledWith(
      '/registries/reg1/artifacts/docker/versions/1.0.1/overview?digest=sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
    )

    const artifactDetailsTab = container.querySelector('div[data-tab-id=artifact_details]')
    await userEvent.click(artifactDetailsTab!)
    expect(mockHistoryPush).toHaveBeenLastCalledWith(
      '/registries/reg1/artifacts/docker/versions/1.0.1/artifact_details?digest=sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
    )

    const sscaTab = container.querySelector('div[data-tab-id=supply_chain]')
    await userEvent.click(sscaTab!)
    expect(mockHistoryPush).toHaveBeenLastCalledWith(
      '/registries/reg1/artifacts/docker/versions/1.0.1/orgs/default/projects/default_project/supply_chain?digest=sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
    )

    const stoTab = container.querySelector('div[data-tab-id=security_tests]')
    await userEvent.click(stoTab!)
    expect(mockHistoryPush).toHaveBeenLastCalledWith(
      '/registries/reg1/artifacts/docker/versions/1.0.1/orgs/default/projects/default_project/security_tests?digest=sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
    )
  })
})
