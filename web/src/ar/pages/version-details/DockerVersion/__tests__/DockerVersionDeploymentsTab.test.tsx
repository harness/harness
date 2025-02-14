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
import { useGetArtifactDeploymentsQuery } from '@harnessio/react-har-service-client'
import { fireEvent, getByTestId, getByText, render, waitFor } from '@testing-library/react'

import { getTableColumn } from '@ar/utils/testUtils/utils'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'

import VersionDetailsPage from '../../VersionDetailsPage'
import {
  mockDockerDeploymentsData,
  mockDockerManifestList,
  mockDockerVersionList,
  mockDockerVersionSummary
} from './__mockData__'

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
  useGetArtifactDeploymentsQuery: jest.fn().mockImplementation(() => ({
    data: { content: mockDockerDeploymentsData },
    error: null,
    isLoading: false,
    refetch: jest.fn()
  }))
}))

describe('Verify docker deployments tab', () => {
  beforeAll(() => {
    jest.clearAllMocks()
  })

  test('should render deployments tab', async () => {
    const { container } = render(
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/artifacts/:artifactIdentifier/versions/:versionIdentifier/:versionTab"
        pathParams={{
          repositoryIdentifier: '1',
          artifactIdentifier: '1',
          versionIdentifier: '1',
          versionTab: 'deployments'
        }}
        queryParams={{
          digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d',
          detailsTab: 'layers'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )
    expect(container).toBeInTheDocument()

    const DockerDeploymentsCard = getByTestId(container, 'docker-deployments-card')
    expect(DockerDeploymentsCard).toBeInTheDocument()

    const dockerIntegrationCard = getByTestId(container, 'integration-deployment-card')
    expect(dockerIntegrationCard).toBeInTheDocument()
  })

  test('verify container card', async () => {
    const { container } = render(
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/artifacts/:artifactIdentifier/versions/:versionIdentifier/:versionTab"
        pathParams={{
          repositoryIdentifier: '1',
          artifactIdentifier: mockDockerVersionSummary.data.imageName,
          versionIdentifier: mockDockerVersionSummary.data.version,
          versionTab: 'deployments'
        }}
        queryParams={{
          digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )
    expect(container).toBeInTheDocument()

    const DockerDeploymentsCard = getByTestId(container, 'docker-deployments-card')
    expect(DockerDeploymentsCard).toBeInTheDocument()
    expect(DockerDeploymentsCard).toHaveTextContent('versionDetails.cards.container.title')
    expect(DockerDeploymentsCard).toHaveTextContent(mockDockerVersionSummary.data.imageName)
    expect(DockerDeploymentsCard).toHaveTextContent('versionDetails.cards.container.versionDigest')
  })

  test('verify deployment card', async () => {
    const { container } = render(
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/artifacts/:artifactIdentifier/versions/:versionIdentifier/:versionTab"
        pathParams={{
          repositoryIdentifier: '1',
          artifactIdentifier: mockDockerVersionSummary.data.imageName,
          versionIdentifier: mockDockerVersionSummary.data.version,
          versionTab: 'deployments'
        }}
        queryParams={{
          digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )
    expect(container).toBeInTheDocument()

    const DockerDeploymentsCard = getByTestId(container, 'integration-deployment-card')
    expect(DockerDeploymentsCard).toBeInTheDocument()
    expect(DockerDeploymentsCard).toHaveTextContent('versionDetails.cards.deploymentsCard.title')
    expect(DockerDeploymentsCard).toHaveTextContent('1')
  })

  test('verify deployment table', async () => {
    render(
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/artifacts/:artifactIdentifier/versions/:versionIdentifier/:versionTab"
        pathParams={{
          repositoryIdentifier: '1',
          artifactIdentifier: mockDockerVersionSummary.data.imageName,
          versionIdentifier: mockDockerVersionSummary.data.version,
          versionTab: 'deployments'
        }}
        queryParams={{
          digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    const getFirstRowColumn = (col: number) => getTableColumn(1, col) as HTMLElement

    const data = mockDockerDeploymentsData.data.deployments.deployments?.[0]
    const envName = getByText(getFirstRowColumn(1), data?.envName as string)
    expect(envName).toBeInTheDocument()

    const type = getByText(getFirstRowColumn(2), 'nonProd')
    expect(type).toBeInTheDocument()

    const infra = getByText(getFirstRowColumn(3), data?.infraName as string)
    expect(infra).toBeInTheDocument()

    const serviceName = getByText(getFirstRowColumn(4), data?.serviceName as string)
    expect(serviceName).toBeInTheDocument()

    const instanceCount = getByText(getFirstRowColumn(5), data?.count?.toString() as string)
    expect(instanceCount).toBeInTheDocument()

    const pipelineName = getByText(getFirstRowColumn(6), data?.lastPipelineExecutionName?.toString() as string)
    expect(pipelineName).toBeInTheDocument()
  })

  test('Pagination should work', async () => {
    const { getByText: getByTextLocal, container } = render(
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/artifacts/:artifactIdentifier/versions/:versionIdentifier/:versionTab"
        pathParams={{
          repositoryIdentifier: '1',
          artifactIdentifier: mockDockerVersionSummary.data.imageName,
          versionIdentifier: mockDockerVersionSummary.data.version,
          versionTab: 'deployments'
        }}
        queryParams={{
          digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    const nextPageBtn = container.querySelector('button[aria-label="Next"]')
    await userEvent.click(nextPageBtn!)

    expect(useGetArtifactDeploymentsQuery).toHaveBeenLastCalledWith({
      artifact: 'maven-app/+',
      queryParams: {
        env_type: undefined,
        page: 1,
        search_term: undefined,
        size: 50,
        sort_field: 'updatedAt',
        sort_order: 'DESC'
      },
      registry_ref: 'undefined/1/+',
      version: '1.0.0'
    })

    const pageSizeSelect = getByTestId(container, 'dropdown-button')
    await userEvent.click(pageSizeSelect)
    const pageSize20option = getByTextLocal('20')
    await userEvent.click(pageSize20option)

    expect(useGetArtifactDeploymentsQuery).toHaveBeenLastCalledWith({
      artifact: 'maven-app/+',
      queryParams: {
        env_type: undefined,
        page: 0,
        search_term: undefined,
        size: 20,
        sort_field: 'updatedAt',
        sort_order: 'DESC'
      },
      registry_ref: 'undefined/1/+',
      version: '1.0.0'
    })
  })

  test('Filters should work', async () => {
    const { getByText: getByTextLocal, getByPlaceholderText } = render(
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/artifacts/:artifactIdentifier/versions/:versionIdentifier/:versionTab"
        pathParams={{
          repositoryIdentifier: '1',
          artifactIdentifier: mockDockerVersionSummary.data.imageName,
          versionIdentifier: mockDockerVersionSummary.data.version,
          versionTab: 'deployments'
        }}
        queryParams={{
          digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    expect(useGetArtifactDeploymentsQuery).toHaveBeenLastCalledWith({
      artifact: 'maven-app/+',
      queryParams: {
        env_type: undefined,
        page: 0,
        search_term: undefined,
        size: 50,
        sort_field: 'updatedAt',
        sort_order: 'DESC'
      },
      registry_ref: 'undefined/1/+',
      version: '1.0.0'
    })
    ;(useGetArtifactDeploymentsQuery as jest.Mock).mockImplementationOnce(() => {
      return {
        data: { content: { data: { deployments: { deployments: [] } } } },
        loading: false,
        error: null,
        refetch: jest.fn()
      }
    })

    const searchInput = getByPlaceholderText('search')
    expect(searchInput).toBeInTheDocument()
    fireEvent.change(searchInput, { target: { value: '1234' } })

    await waitFor(async () => {
      expect(useGetArtifactDeploymentsQuery).toHaveBeenCalledWith({
        artifact: 'maven-app/+',
        queryParams: {
          env_type: undefined,
          page: 0,
          search_term: '1234',
          size: 50,
          sort_field: 'updatedAt',
          sort_order: 'DESC'
        },
        registry_ref: 'undefined/1/+',
        version: '1.0.0'
      })
    })

    const clearAllFiltersBtn = getByTextLocal('clearFilters')
    await userEvent.click(clearAllFiltersBtn)

    expect(useGetArtifactDeploymentsQuery).toHaveBeenLastCalledWith({
      artifact: 'maven-app/+',
      queryParams: {
        env_type: undefined,
        search_term: '',
        page: 0,
        size: 50,
        sort_field: 'updatedAt',
        sort_order: 'DESC'
      },
      registry_ref: 'undefined/1/+',
      version: '1.0.0'
    })
  })

  test('Filters should work', async () => {
    const { getByText: getByTextLocal } = render(
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/artifacts/:artifactIdentifier/versions/:versionIdentifier/:versionTab"
        pathParams={{
          repositoryIdentifier: '1',
          artifactIdentifier: mockDockerVersionSummary.data.imageName,
          versionIdentifier: mockDockerVersionSummary.data.version,
          versionTab: 'deployments'
        }}
        queryParams={{
          digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    const evnName = getByTextLocal('versionDetails.deploymentsTable.columns.environment').nextSibling
      ?.firstChild as HTMLElement
    await userEvent.click(evnName)
    expect(useGetArtifactDeploymentsQuery).toHaveBeenLastCalledWith({
      artifact: 'maven-app/+',
      queryParams: {
        env_type: undefined,
        page: 0,
        search_term: undefined,
        size: 50,
        sort_field: 'envName',
        sort_order: 'ASC'
      },
      registry_ref: 'undefined/1/+',
      version: '1.0.0'
    })

    await userEvent.click(evnName)

    expect(useGetArtifactDeploymentsQuery).toHaveBeenLastCalledWith({
      artifact: 'maven-app/+',
      queryParams: {
        env_type: undefined,
        page: 0,
        search_term: undefined,
        size: 50,
        sort_field: 'envName',
        sort_order: 'DESC'
      },
      registry_ref: 'undefined/1/+',
      version: '1.0.0'
    })
  })

  test('verify deployment table for api failure', async () => {
    const refetchFn = jest.fn()
    ;(useGetArtifactDeploymentsQuery as jest.Mock).mockImplementation(() => ({
      data: null,
      error: { message: 'error message' },
      isLoading: false,
      refetch: refetchFn
    }))
    const { container } = render(
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/artifacts/:artifactIdentifier/versions/:versionIdentifier/:versionTab"
        pathParams={{
          repositoryIdentifier: '1',
          artifactIdentifier: mockDockerVersionSummary.data.imageName,
          versionIdentifier: mockDockerVersionSummary.data.version,
          versionTab: 'deployments'
        }}
        queryParams={{
          digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    expect(getByText(container, 'error message')).toBeInTheDocument()
    const retryBtn = container.querySelector('button[aria-label=Retry]')
    expect(retryBtn).toBeInTheDocument()
    await userEvent.click(retryBtn!)
    await waitFor(() => {
      expect(refetchFn).toHaveBeenCalled()
    })
  })
})
