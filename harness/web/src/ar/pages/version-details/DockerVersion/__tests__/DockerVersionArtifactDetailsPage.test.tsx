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
import copy from 'clipboard-copy'
import userEvent from '@testing-library/user-event'
import { getByTestId, getByText, render, waitFor } from '@testing-library/react'
import { useGetDockerArtifactLayersQuery, useGetDockerArtifactManifestQuery } from '@harnessio/react-har-service-client'

import { getTableColumn } from '@ar/utils/testUtils/utils'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'

import VersionDetailsPage from '../../VersionDetailsPage'
import {
  mockDockerArtifactLayers,
  mockDockerArtifactManifest,
  mockDockerManifestList,
  mockDockerVersionList,
  mockDockerVersionSummary
} from './__mockData__'

const mockHistoryPush = jest.fn()
// eslint-disable-next-line jest-no-mock
jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useHistory: () => ({
    push: mockHistoryPush
  })
}))

jest.mock('clipboard-copy', () => ({
  __esModule: true,
  default: jest.fn()
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
  useGetDockerArtifactLayersQuery: jest.fn().mockImplementation(() => ({
    data: { content: mockDockerArtifactLayers },
    error: null,
    isLoading: false,
    refetch: jest.fn()
  })),
  useGetDockerArtifactManifestQuery: jest.fn().mockImplementation(() => ({
    data: { content: mockDockerArtifactManifest },
    error: null,
    isLoading: false,
    refetch: jest.fn()
  }))
}))

describe('Verify Docker Version Artifact Details Tab', () => {
  beforeAll(() => {
    jest.clearAllMocks()
  })

  test('Verify Sub Tabs', async () => {
    const { container } = render(
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/artifacts/:artifactIdentifier/versions/:versionIdentifier/:versionTab"
        pathParams={{
          repositoryIdentifier: '1',
          artifactIdentifier: '1',
          versionIdentifier: '1',
          versionTab: 'artifact_details'
        }}
        queryParams={{
          digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d',
          detailsTab: 'layers'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    expect(getByTestId(container, 'layers')).toBeInTheDocument()
    expect(getByTestId(container, 'manifest')).toBeInTheDocument()

    const manifestTab = getByTestId(container, 'manifest')
    await userEvent.click(manifestTab)
    expect(mockHistoryPush).toHaveBeenCalledWith(
      '/registries/1/artifacts/1/versions/1/artifact_details?digest=sha256%3A144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d&detailsTab=manifest'
    )

    const layersTab = getByTestId(container, 'layers')
    await userEvent.click(layersTab)
    expect(mockHistoryPush).toHaveBeenCalledWith(
      '/registries/1/artifacts/1/versions/1/artifact_details?digest=sha256%3A144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d&detailsTab=layers'
    )
  })

  test('Verify Layers Tab', async () => {
    const { container } = render(
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/artifacts/:artifactIdentifier/versions/:versionIdentifier/:versionTab"
        pathParams={{
          repositoryIdentifier: '1',
          artifactIdentifier: '1',
          versionIdentifier: '1',
          versionTab: 'artifact_details'
        }}
        queryParams={{
          digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d',
          detailsTab: 'layers'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    expect(container).toHaveTextContent('versionDetails.artifactDetails.layers.imageLayers')

    const getTableRowColumn = (row: number, col: number) => getTableColumn(row, col) as HTMLElement

    const data = mockDockerArtifactLayers.data.layers?.[0]
    const row = 1
    expect(getTableRowColumn(row, 1)).toHaveTextContent(row.toString())
    expect(getTableRowColumn(row, 2)).toHaveTextContent(data?.command as string)
    expect(getTableRowColumn(row, 3)).toHaveTextContent(data?.size as string)
    const copyColumn = getTableRowColumn(row, 4)
    const copyBtn = copyColumn.querySelector('[data-icon="code-copy"]') as HTMLElement
    expect(copyBtn).toBeInTheDocument()
    await userEvent.click(copyBtn)
    expect(copy).toHaveBeenCalledWith(data?.command)
  })

  test('should show error message if failed to load layers data', async () => {
    const refetchFn = jest.fn()
    ;(useGetDockerArtifactLayersQuery as jest.Mock).mockImplementation(() => ({
      data: null,
      error: { message: 'Failed to fetch artifact details' },
      isLoading: false,
      refetch: refetchFn
    }))
    const { container } = render(
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/artifacts/:artifactIdentifier/versions/:versionIdentifier/:versionTab"
        pathParams={{
          repositoryIdentifier: '1',
          artifactIdentifier: '1',
          versionIdentifier: '1',
          versionTab: 'artifact_details'
        }}
        queryParams={{
          digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d',
          detailsTab: 'layers'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )
    expect(getByText(container, 'Failed to fetch artifact details')).toBeInTheDocument()
    const retryBtn = container.querySelector('button[aria-label=Retry]')
    expect(retryBtn).toBeInTheDocument()
    await userEvent.click(retryBtn!)
    await waitFor(() => {
      expect(refetchFn).toHaveBeenCalled()
    })
  })

  test('Verify Manifest Tab', async () => {
    const { container } = render(
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/artifacts/:artifactIdentifier/versions/:versionIdentifier/:versionTab"
        pathParams={{
          repositoryIdentifier: '1',
          artifactIdentifier: '1',
          versionIdentifier: '1',
          versionTab: 'artifact_details'
        }}
        queryParams={{
          digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d',
          detailsTab: 'manifest'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    expect(container).toHaveTextContent('versionDetails.artifactDetails.tabs.manifest')

    const copyBtn = container.querySelector('[data-icon="code-copy"]') as HTMLElement
    expect(copyBtn).toBeInTheDocument()
    await userEvent.click(copyBtn)
    expect(copy).toHaveBeenCalledWith(mockDockerArtifactManifest.data.manifest)
  })

  test('should show error message if failed to load manifest data', async () => {
    const refetchFn = jest.fn()
    ;(useGetDockerArtifactManifestQuery as jest.Mock).mockImplementation(() => ({
      data: null,
      error: { message: 'Failed to fetch artifact details' },
      isLoading: false,
      refetch: refetchFn
    }))
    const { container } = render(
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/artifacts/:artifactIdentifier/versions/:versionIdentifier/:versionTab"
        pathParams={{
          repositoryIdentifier: '1',
          artifactIdentifier: '1',
          versionIdentifier: '1',
          versionTab: 'artifact_details'
        }}
        queryParams={{
          digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d',
          detailsTab: 'manifest'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )
    expect(getByText(container, 'Failed to fetch artifact details')).toBeInTheDocument()
    const retryBtn = container.querySelector('button[aria-label=Retry]')
    expect(retryBtn).toBeInTheDocument()
    await userEvent.click(retryBtn!)
    await waitFor(() => {
      expect(refetchFn).toHaveBeenCalled()
    })
  })
})
