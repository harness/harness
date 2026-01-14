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
import { getByTestId, getByText, render, waitFor } from '@testing-library/react'
import { useGetDockerArtifactDetailsQuery } from '@harnessio/react-har-service-client'

import '@ar/pages/version-details/VersionFactory'
import '@ar/pages/repository-details/RepositoryFactory'

import { Parent } from '@ar/common/types'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'

import {
  mockDockerArtifactDetails,
  mockDockerArtifactLayers,
  mockDockerArtifactManifest,
  mockDockerManifestList,
  mockDockerVersionList,
  mockDockerVersionSummary
} from './__mockData__'
import OSSVersionDetailsPage from '../../OSSVersionDetailsPage'

jest.mock('@harnessio/react-har-service-client', () => ({
  useGetArtifactVersionSummaryQuery: jest.fn().mockImplementation(() => ({
    data: { content: mockDockerVersionSummary },
    error: null,
    isLoading: false,
    refetch: jest.fn()
  })),
  useGetDockerArtifactDetailsQuery: jest.fn().mockImplementation(() => ({
    data: { content: mockDockerArtifactDetails },
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

describe('verify docker version details page for oss', () => {
  test('Should render content without error', () => {
    render(
      <ArTestWrapper
        parent={Parent.OSS}
        path="/registries/:repositoryIdentifier/artifacts/:artifactIdentifier/versions/:versionIdentifier"
        pathParams={{
          repositoryIdentifier: '1',
          artifactIdentifier: '1',
          versionIdentifier: '1'
        }}
        queryParams={{
          digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
        }}>
        <OSSVersionDetailsPage />
      </ArTestWrapper>
    )
    const versionDetailsPage = document.querySelector('[data-testid="version-details-page-oss"]') as HTMLElement
    expect(versionDetailsPage).toBeInTheDocument()

    const pageHeader = getByTestId(versionDetailsPage, 'page-header')
    expect(pageHeader).toBeInTheDocument()

    const generalInfoCard = getByTestId(versionDetailsPage, 'general-information-card')
    expect(generalInfoCard).toBeInTheDocument()
  })

  test('Should render error message if failed to load general info', async () => {
    const refetchFn = jest.fn()
    ;(useGetDockerArtifactDetailsQuery as jest.Mock).mockImplementation(() => ({
      data: null,
      error: { message: 'Failed to load general info' },
      isLoading: false,
      refetch: refetchFn
    }))
    render(
      <ArTestWrapper
        parent={Parent.OSS}
        path="/registries/:repositoryIdentifier/artifacts/:artifactIdentifier/versions/:versionIdentifier"
        pathParams={{
          repositoryIdentifier: '1',
          artifactIdentifier: '1',
          versionIdentifier: '1'
        }}
        queryParams={{
          digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
        }}>
        <OSSVersionDetailsPage />
      </ArTestWrapper>
    )
    const versionDetailsPage = document.querySelector('[data-testid="version-details-page-oss"]') as HTMLElement
    expect(getByText(versionDetailsPage, 'Failed to load general info')).toBeInTheDocument()
    const retryBtn = versionDetailsPage.querySelector('button[aria-label=Retry]')
    expect(retryBtn).toBeInTheDocument()
    await userEvent.click(retryBtn!)
    await waitFor(() => {
      expect(refetchFn).toHaveBeenCalled()
    })
  })
})
