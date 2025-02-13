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
import {
  useGetDockerArtifactDetailsQuery,
  useGetDockerArtifactIntegrationDetailsQuery
} from '@harnessio/react-har-service-client'

import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'

import '@ar/pages/version-details/VersionFactory'
import '@ar/pages/repository-details/RepositoryFactory'
import { DEFAULT_DATE_TIME_FORMAT } from '@ar/constants'
import { getReadableDateTime } from '@ar/common/dateUtils'

import VersionDetailsPage from '../../VersionDetailsPage'
import {
  mockDockerArtifactDetails,
  mockDockerArtifactIntegrationDetails,
  mockDockerManifestList,
  mockDockerVersionList,
  mockDockerVersionSummary
} from './__mockData__'

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
  }))
}))

describe('Verify docker version overview page', () => {
  beforeAll(() => {
    jest.clearAllMocks()
  })

  test('Should render integration detail cards without error', async () => {
    const { container } = render(
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/artifacts/:artifactIdentifier/versions/:versionIdentifier/:versionTab"
        pathParams={{
          repositoryIdentifier: '1',
          artifactIdentifier: '1',
          versionIdentifier: '1',
          versionTab: 'overview'
        }}
        queryParams={{ digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d' }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    expect(getByTestId(container, 'integration-cards')).toBeInTheDocument()

    const data = mockDockerArtifactIntegrationDetails.data
    const deploymentCard = getByTestId(container, 'integration-deployment-card')
    expect(deploymentCard).toBeInTheDocument()
    expect(deploymentCard).toHaveTextContent(data.deploymentsDetails?.totalDeployment?.toString() as string)
    expect(deploymentCard).toHaveTextContent(data.buildDetails?.pipelineDisplayName as string)
    expect(deploymentCard).toHaveTextContent(data.buildDetails?.pipelineExecutionId as string)

    const sscaCard = getByTestId(container, 'integration-supply-chain-card')
    expect(sscaCard).toBeInTheDocument()
    expect(sscaCard).toHaveTextContent(data.sbomDetails?.componentsCount?.toString() as string)
    expect(sscaCard).toHaveTextContent(Number(data.sbomDetails?.avgScore).toFixed(2) as string)
    expect(getByText(sscaCard, 'versionDetails.cards.supplyChain.downloadSbom')).toBeInTheDocument()

    const securityTestsCard = getByTestId(container, 'integration-security-tests-card')
    expect(securityTestsCard).toBeInTheDocument()
    expect(securityTestsCard).toHaveTextContent(data.stoDetails?.total?.toString() as string)
    expect(securityTestsCard).toHaveTextContent(data.stoDetails?.critical?.toString() as string)
    expect(securityTestsCard).toHaveTextContent(data.stoDetails?.high?.toString() as string)
    expect(securityTestsCard).toHaveTextContent(data.stoDetails?.medium?.toString() as string)
    expect(securityTestsCard).toHaveTextContent(data.stoDetails?.low?.toString() as string)
  })

  test('should show error message if failed to load integration data', async () => {
    const refetchFn = jest.fn()
    ;(useGetDockerArtifactIntegrationDetailsQuery as jest.Mock).mockImplementation(() => ({
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
          versionTab: 'overview'
        }}
        queryParams={{ digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d' }}>
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

  test('should render overview content without error', async () => {
    const { container } = render(
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/artifacts/:artifactIdentifier/versions/:versionIdentifier/:versionTab"
        pathParams={{
          repositoryIdentifier: '1',
          artifactIdentifier: '1',
          versionIdentifier: '1',
          versionTab: 'overview'
        }}
        queryParams={{ digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d' }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    const generalInformationCard = getByTestId(container, 'general-information-card')
    expect(generalInformationCard).toBeInTheDocument()

    const data = mockDockerArtifactDetails.data
    expect(generalInformationCard).toHaveTextContent('versionDetails.overview.generalInformation.title')

    expect(generalInformationCard).toHaveTextContent('versionDetails.overview.generalInformation.name' + data.imageName)
    expect(generalInformationCard).toHaveTextContent(
      'versionDetails.overview.generalInformation.version' + data.version
    )
    expect(generalInformationCard).toHaveTextContent(
      'versionDetails.overview.generalInformation.packageType' + 'packageTypes.dockerPackage'
    )
    expect(generalInformationCard).toHaveTextContent(
      'versionDetails.overview.generalInformation.digest' +
        'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
    )
    expect(generalInformationCard).toHaveTextContent('versionDetails.overview.generalInformation.size' + data.size)
    expect(generalInformationCard).toHaveTextContent(
      'versionDetails.overview.generalInformation.downloads' + data.downloadsCount
    )
    expect(generalInformationCard).toHaveTextContent(
      'versionDetails.overview.generalInformation.uploadedBy' +
        getReadableDateTime(Number(data.modifiedAt), DEFAULT_DATE_TIME_FORMAT)
    )
    expect(generalInformationCard).toHaveTextContent(
      'versionDetails.overview.generalInformation.pullCommand' + data.pullCommand
    )
  })

  test('should show error message if failed to load overview data', async () => {
    const refetchFn = jest.fn()
    ;(useGetDockerArtifactDetailsQuery as jest.Mock).mockImplementation(() => ({
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
          versionTab: 'overview'
        }}
        queryParams={{ digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d' }}>
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
