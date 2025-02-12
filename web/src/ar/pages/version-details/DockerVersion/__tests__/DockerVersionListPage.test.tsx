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
import { fireEvent, getByText, render, waitFor } from '@testing-library/react'
import {
  useGetAllArtifactVersionsQuery as _useGetAllArtifactVersionsQuery,
  useGetDockerArtifactManifestsQuery as _useGetDockerArtifactManifestsQuery,
  ArtifactVersionMetadata,
  DockerManifestDetails
} from '@harnessio/react-har-service-client'

import '@ar/pages/version-details/VersionFactory'
import '@ar/pages/repository-details/RepositoryFactory'

import { getShortDigest } from '@ar/pages/digest-list/utils'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import VersionListPage from '@ar/pages/version-list/VersionListPage'
import { getTableColumn, getTableRow } from '@ar/utils/testUtils/utils'

import { mockDockerLatestVersionListTableData, mockDockerManifestListTableData } from './__mockData__'

const useGetAllArtifactVersionsQuery = _useGetAllArtifactVersionsQuery as jest.Mock
const useGetDockerArtifactManifestsQuery = _useGetDockerArtifactManifestsQuery as jest.Mock

jest.mock('@harnessio/react-har-service-client', () => ({
  useGetAllArtifactVersionsQuery: jest.fn(),
  useGetDockerArtifactManifestsQuery: jest.fn()
}))

jest.mock('clipboard-copy', () => ({
  __esModule: true,
  default: jest.fn()
}))

describe('Verify DockerVersion List Page', () => {
  test('Should render empty list text if response is empty', () => {
    useGetAllArtifactVersionsQuery.mockImplementation(() => {
      return {
        data: { content: { data: [] } },
        loading: false,
        error: null,
        refetch: jest.fn()
      }
    })

    const { getByText: getByTextLocal } = render(
      <ArTestWrapper>
        <VersionListPage packageType="DOCKER" />
      </ArTestWrapper>
    )

    const noItemsText = getByTextLocal('versionList.table.noVersionsTitle')
    expect(noItemsText).toBeInTheDocument()
  })

  test('Should render docker version list', async () => {
    useGetAllArtifactVersionsQuery.mockImplementation(() => {
      return {
        data: { content: { data: mockDockerLatestVersionListTableData } },
        loading: false,
        error: null,
        refetch: jest.fn()
      }
    })
    const { container, getByTestId } = render(
      <ArTestWrapper>
        <VersionListPage packageType="DOCKER" />
      </ArTestWrapper>
    )

    const artifact = mockDockerLatestVersionListTableData.artifactVersions?.[0] as ArtifactVersionMetadata

    const table = container.querySelector('[class*="TableV2--table"]')
    expect(table).toBeInTheDocument()

    const rows = container.querySelectorAll('[class*="TableV2--row"]')
    expect(rows).toHaveLength(1)

    const getFirstRowColumn = (col: number) => getTableColumn(1, col) as HTMLElement

    const name = getByText(getFirstRowColumn(2), artifact.name as string)
    expect(name).toBeInTheDocument()

    const nonProdDep = getByTestId('nonProdDeployments')
    const prodDep = getByTestId('prodDeployments')

    expect(getByText(nonProdDep, artifact.deploymentMetadata?.nonProdEnvCount as number)).toBeInTheDocument()
    expect(getByText(nonProdDep, 'nonProd')).toBeInTheDocument()

    expect(getByText(prodDep, artifact.deploymentMetadata?.prodEnvCount as number)).toBeInTheDocument()
    expect(getByText(prodDep, 'prod')).toBeInTheDocument()

    const digestValue = getByText(getFirstRowColumn(4), artifact.digestCount as number)
    expect(digestValue).toBeInTheDocument()

    const curlColumn = getFirstRowColumn(6)
    expect(curlColumn).toHaveTextContent('copy')
    const copyCurlBtn = curlColumn.querySelector('[data-icon="code-copy"]') as HTMLElement
    expect(copyCurlBtn).toBeInTheDocument()
    await userEvent.click(copyCurlBtn)
    expect(copy).toHaveBeenCalled()

    const expandIcon = getFirstRowColumn(1).querySelector('[data-icon="chevron-down"') as HTMLElement
    expect(expandIcon).toBeInTheDocument()
  })

  test('Should render docker manifest list', async () => {
    useGetAllArtifactVersionsQuery.mockImplementation(() => {
      return {
        data: { content: { data: mockDockerLatestVersionListTableData } },
        loading: false,
        error: null,
        refetch: jest.fn()
      }
    })
    useGetDockerArtifactManifestsQuery.mockImplementation(() => {
      return {
        data: { content: mockDockerManifestListTableData },
        loading: false,
        error: null,
        refetch: jest.fn()
      }
    })

    const { container } = render(
      <ArTestWrapper>
        <VersionListPage packageType="DOCKER" />
      </ArTestWrapper>
    )

    const getFirstRowColumn = (col: number) => getTableColumn(1, col, container) as HTMLElement

    const expandIcon = getFirstRowColumn(1).querySelector('[data-icon="chevron-down"') as HTMLElement
    expect(expandIcon).toBeInTheDocument()
    await userEvent.click(expandIcon)

    const digestTableData = mockDockerManifestListTableData.data.manifests?.[0] as DockerManifestDetails
    const digestListTable = getTableRow(1, container) as HTMLElement
    const getFirstDigestRowColumn = (col: number) => getTableColumn(1, col, digestListTable) as HTMLElement

    const digestName = getFirstDigestRowColumn(1)
    expect(digestName).toHaveTextContent(getShortDigest(digestTableData.digest))

    const osArch = getFirstDigestRowColumn(2)
    expect(osArch).toHaveTextContent(digestTableData.osArch)

    const size = getFirstDigestRowColumn(3)
    expect(size).toHaveTextContent(digestTableData.size as string)

    const downloads = getFirstDigestRowColumn(5)
    expect(downloads).toHaveTextContent(digestTableData.downloadsCount?.toString() as string)
  })

  test('Should show error message docker manifest list ', async () => {
    useGetAllArtifactVersionsQuery.mockImplementation(() => {
      return {
        data: { content: { data: mockDockerLatestVersionListTableData } },
        loading: false,
        error: null,
        refetch: jest.fn()
      }
    })
    const mockRefetchFn = jest.fn().mockImplementation(() => undefined)

    useGetDockerArtifactManifestsQuery.mockImplementation(() => {
      return {
        data: null,
        loading: false,
        error: { message: 'failed to load data' },
        refetch: mockRefetchFn
      }
    })

    const { container, getByText: getByTextLocal } = render(
      <ArTestWrapper>
        <VersionListPage packageType="DOCKER" />
      </ArTestWrapper>
    )

    const getFirstRowColumn = (col: number) => getTableColumn(1, col, container) as HTMLElement

    const expandIcon = getFirstRowColumn(1).querySelector('[data-icon="chevron-down"') as HTMLElement
    expect(expandIcon).toBeInTheDocument()
    await userEvent.click(expandIcon)

    const errorText = getByTextLocal('failed to load data')
    expect(errorText).toBeInTheDocument()

    const retryBtn = getByTextLocal('Retry')
    expect(retryBtn).toBeInTheDocument()

    await userEvent.click(retryBtn)
    expect(mockRefetchFn).toHaveBeenCalled()
  })

  test('Pagination should work', async () => {
    useGetAllArtifactVersionsQuery.mockImplementation(() => {
      return {
        data: { content: { data: mockDockerLatestVersionListTableData } },
        loading: false,
        error: null,
        refetch: jest.fn()
      }
    })
    const { getByText: getByTextLocal, getByTestId } = render(
      <ArTestWrapper>
        <VersionListPage packageType="DOCKER" />
      </ArTestWrapper>
    )

    const nextPageBtn = getByTextLocal('Next')
    await userEvent.click(nextPageBtn)

    expect(useGetAllArtifactVersionsQuery).toHaveBeenLastCalledWith({
      artifact: 'undefined/+',
      queryParams: { page: 1, search_term: '', size: 50, sort_field: 'updatedAt', sort_order: 'DESC' },
      registry_ref: 'undefined/+',
      stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
    })

    const pageSizeSelect = getByTestId('dropdown-button')
    await userEvent.click(pageSizeSelect)
    const pageSize20option = getByTextLocal('20')
    await userEvent.click(pageSize20option)

    expect(useGetAllArtifactVersionsQuery).toHaveBeenLastCalledWith({
      artifact: 'undefined/+',
      queryParams: { page: 0, search_term: '', size: 20, sort_field: 'updatedAt', sort_order: 'DESC' },
      registry_ref: 'undefined/+',
      stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
    })
  })

  test('Filters should work', async () => {
    useGetAllArtifactVersionsQuery.mockImplementation(() => {
      return {
        data: { content: { data: mockDockerLatestVersionListTableData } },
        loading: false,
        error: null,
        refetch: jest.fn()
      }
    })
    const { getByText: getByTextLocal, getByPlaceholderText } = render(
      <ArTestWrapper>
        <VersionListPage packageType="DOCKER" />
      </ArTestWrapper>
    )

    expect(useGetAllArtifactVersionsQuery).toHaveBeenLastCalledWith({
      artifact: 'undefined/+',
      queryParams: { page: 0, search_term: '', size: 50, sort_field: 'updatedAt', sort_order: 'DESC' },
      registry_ref: 'undefined/+',
      stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
    })

    useGetAllArtifactVersionsQuery.mockImplementationOnce(() => {
      return {
        data: { content: { data: { artifactVersions: [] } } },
        loading: false,
        error: null,
        refetch: jest.fn()
      }
    })

    const searchInput = getByPlaceholderText('search')
    expect(searchInput).toBeInTheDocument()
    fireEvent.change(searchInput, { target: { value: '1234' } })

    await waitFor(async () => {
      expect(useGetAllArtifactVersionsQuery).toHaveBeenLastCalledWith({
        artifact: 'undefined/+',
        queryParams: { page: 0, search_term: '1234', size: 50, sort_field: 'updatedAt', sort_order: 'DESC' },
        registry_ref: 'undefined/+',
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      })
    })

    const clearAllFiltersBtn = getByTextLocal('clearFilters')
    await userEvent.click(clearAllFiltersBtn)

    expect(useGetAllArtifactVersionsQuery).toHaveBeenLastCalledWith({
      artifact: 'undefined/+',
      queryParams: { page: 0, search_term: '', size: 50, sort_field: 'updatedAt', sort_order: 'DESC' },
      registry_ref: 'undefined/+',
      stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
    })
  })

  test('Sorting should work', async () => {
    const { getByText: getByTextLocal } = render(
      <ArTestWrapper>
        <VersionListPage packageType="DOCKER" />
      </ArTestWrapper>
    )

    const artifactNameSortIcon = getByTextLocal('versionList.table.columns.version').nextSibling
      ?.firstChild as HTMLElement
    await userEvent.click(artifactNameSortIcon)
    expect(useGetAllArtifactVersionsQuery).toHaveBeenLastCalledWith({
      artifact: 'undefined/+',
      queryParams: { page: 0, search_term: '', size: 50, sort_field: 'name', sort_order: 'ASC' },
      registry_ref: 'undefined/+',
      stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
    })
  })

  test('Should show error message with which listing api fails', async () => {
    const mockRefetchFn = jest.fn().mockImplementation(() => undefined)
    useGetAllArtifactVersionsQuery.mockImplementationOnce(() => {
      return {
        data: null,
        loading: false,
        error: { message: 'error message' },
        refetch: mockRefetchFn
      }
    })

    const { getByText: getByTextLocal } = render(
      <ArTestWrapper>
        <VersionListPage packageType="DOCKER" />
      </ArTestWrapper>
    )
    const errorText = getByTextLocal('error message')
    expect(errorText).toBeInTheDocument()

    const retryBtn = getByTextLocal('Retry')
    expect(retryBtn).toBeInTheDocument()

    await userEvent.click(retryBtn)
    expect(mockRefetchFn).toHaveBeenCalled()
  })
})
