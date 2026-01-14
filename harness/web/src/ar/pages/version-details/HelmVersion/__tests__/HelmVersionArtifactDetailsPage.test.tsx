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
import { getByText, render, waitFor } from '@testing-library/react'
import { useGetHelmArtifactManifestQuery } from '@harnessio/react-har-service-client'

import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'

import { prettifyManifestJSON } from '../../utils'
import VersionDetailsPage from '../../VersionDetailsPage'
import { mockHelmArtifactManifest, mockHelmVersionList, mockHelmVersionSummary } from './__mockData__'

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
    data: { content: mockHelmVersionSummary },
    error: null,
    isLoading: false,
    refetch: jest.fn()
  })),
  getAllArtifactVersions: jest.fn().mockImplementation(
    () =>
      new Promise(success => {
        success({ content: mockHelmVersionList })
      })
  ),
  useGetHelmArtifactManifestQuery: jest.fn().mockImplementation(() => ({
    data: { content: mockHelmArtifactManifest },
    error: null,
    isLoading: false,
    refetch: jest.fn()
  }))
}))

describe('Verify Helm Version Artifact Details Tab', () => {
  beforeAll(() => {
    jest.clearAllMocks()
  })

  test('Verify Manifest', async () => {
    const { container } = render(
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/artifacts/:artifactIdentifier/versions/:versionIdentifier/:versionTab"
        pathParams={{
          repositoryIdentifier: '1',
          artifactIdentifier: '1',
          versionIdentifier: '1',
          versionTab: 'artifact_details'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    expect(container).toHaveTextContent('versionDetails.artifactDetails.tabs.manifest')

    const copyBtn = container.querySelector('[data-icon="code-copy"]') as HTMLElement
    expect(copyBtn).toBeInTheDocument()
    await userEvent.click(copyBtn)
    expect(copy).toHaveBeenCalledWith(prettifyManifestJSON(mockHelmArtifactManifest.data.manifest))
  })

  test('should show error message if failed to load manifest data', async () => {
    const refetchFn = jest.fn()
    ;(useGetHelmArtifactManifestQuery as jest.Mock).mockImplementation(() => ({
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
