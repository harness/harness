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
import { downloadSbom } from '@harnessio/react-ssca-manager-client'

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
  mockDockerSbomData,
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

const showErrorToast = jest.fn()

jest.mock('@harnessio/uicore', () => ({
  ...jest.requireActual('@harnessio/uicore'),
  useToaster: jest.fn().mockImplementation(() => ({
    showSuccess: jest.fn(),
    showError: showErrorToast,
    clear: jest.fn()
  }))
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
  }))
}))

jest.mock('@harnessio/react-ssca-manager-client', () => ({
  downloadSbom: jest.fn().mockImplementation(
    () =>
      new Promise(success => {
        success({ content: mockDockerSbomData })
      })
  )
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
    await userEvent.click(deploymentCard)
    await waitFor(() => {
      expect(mockHistoryPush).toHaveBeenCalledWith(
        '/registries/1/artifacts/1/versions/1/deployments?digest=sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
      )
    })

    const sscaCard = getByTestId(container, 'integration-supply-chain-card')
    expect(sscaCard).toBeInTheDocument()
    expect(sscaCard).toHaveTextContent(data.sbomDetails?.componentsCount?.toString() as string)
    expect(sscaCard).toHaveTextContent(Number(data.sbomDetails?.avgScore).toFixed(2) as string)
    expect(getByText(sscaCard, 'versionDetails.cards.supplyChain.downloadSbom')).toBeInTheDocument()
    await userEvent.click(sscaCard)
    await waitFor(() => {
      expect(mockHistoryPush).toHaveBeenCalledWith(
        '/registries/1/artifacts/1/versions/1/artifact-sources/67a5dccf6d75916b0c3ea1b5/artifacts/67a5dccf6d75916b0c3ea1b6/supply_chain?digest=sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
      )
    })

    const securityTestsCard = getByTestId(container, 'integration-security-tests-card')
    expect(securityTestsCard).toBeInTheDocument()
    expect(securityTestsCard).toHaveTextContent(data.stoDetails?.total?.toString() as string)
    expect(securityTestsCard).toHaveTextContent(data.stoDetails?.critical?.toString() as string)
    expect(securityTestsCard).toHaveTextContent(data.stoDetails?.high?.toString() as string)
    expect(securityTestsCard).toHaveTextContent(data.stoDetails?.medium?.toString() as string)
    expect(securityTestsCard).toHaveTextContent(data.stoDetails?.low?.toString() as string)
    await userEvent.click(securityTestsCard)
    await waitFor(() => {
      expect(mockHistoryPush).toHaveBeenCalledWith(
        '/registries/1/artifacts/1/versions/1/pipelines/HARNESS_ARTIFACT_SCAN_PIPELINE/executions/Tbi7s6nETjmOMKU3Qrnm7A/security_tests?digest=sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
      )
    })
  })

  test('Verify action button on ssca card', async () => {
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

    const sscaCard = getByTestId(container, 'integration-supply-chain-card')
    const downloadSbomBtn = getByText(sscaCard, 'versionDetails.cards.supplyChain.downloadSbom')
    expect(downloadSbomBtn).toBeInTheDocument()
    await userEvent.click(downloadSbomBtn)
    await waitFor(() => {
      expect(downloadSbom).toHaveBeenCalledWith({ 'orchestration-id': 'yw0D70fiTqetxx0HIyvEUQ', org: '', project: '' })
    })

    // empty response scenario
    ;(downloadSbom as jest.Mock).mockImplementationOnce(
      () =>
        new Promise(success => {
          success({ content: { sbom: null } })
        })
    )
    await userEvent.click(downloadSbomBtn)
    await waitFor(() => {
      expect(showErrorToast).toHaveBeenCalledWith('versionDetails.cards.supplyChain.SBOMDataNotAvailable')
    })

    // error scenario
    ;(downloadSbom as jest.Mock).mockImplementationOnce(
      () =>
        new Promise((_, reject) => {
          reject({ message: 'error message' })
        })
    )
    await userEvent.click(downloadSbomBtn)
    await waitFor(() => {
      expect(showErrorToast).toHaveBeenCalledWith('error message')
    })
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
