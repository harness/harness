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
import { fireEvent, getByText, render, waitFor } from '@testing-library/react'
import {
  useGetAllArtifactVersionsQuery as _useGetAllArtifactVersionsQuery,
  ArtifactVersionMetadata,
  useGetDockerArtifactManifestsQuery as _useGetDockerArtifactManifestsQuery
} from '@harnessio/react-har-service-client'
import userEvent from '@testing-library/user-event'
import repositoryFactory from '@ar/frameworks/RepositoryStep/RepositoryFactory'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import { DockerRepositoryType } from '@ar/pages/repository-details/DockerRepository/DockerRepositoryType'
import { DockerVersionType } from '@ar/pages/version-details/DockerVersion/DockerVersionType'
import versionFactory from '@ar/frameworks/Version/VersionFactory'
import { handleToggleExpandableRow } from '@ar/components/TableCells/utils'
import { getTableColumn } from '@ar/utils/testUtils/utils'
import VersionListPage from '../VersionListPage'
import {
  mockEmptyUseGetAllArtifactVersionsQueryResponse,
  mockErrorUseGetAllArtifactVersionsQueryResponse,
  mockDockerUseGetAllArtifactVersionsQueryResponse,
  mockUseGetDockerArtifactManifestsQueryResponse
} from './__mockData__'

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

jest.mock('@ar/components/TableCells/utils', () => ({
  handleToggleExpandableRow: jest.fn().mockImplementation(() => {
    return (val: Set<any>) => new Set(val)
  })
}))

describe('Verify Version List Page', () => {
  beforeAll(() => {
    repositoryFactory.registerStep(new DockerRepositoryType())
    versionFactory.registerStep(new DockerVersionType())

    useGetAllArtifactVersionsQuery.mockImplementation(() => {
      return mockDockerUseGetAllArtifactVersionsQueryResponse
    })

    useGetDockerArtifactManifestsQuery.mockImplementation(() => {
      return mockUseGetDockerArtifactManifestsQueryResponse
    })
  })

  test('Should render empty list text if response is empty', () => {
    useGetAllArtifactVersionsQuery.mockImplementationOnce(() => {
      return mockEmptyUseGetAllArtifactVersionsQueryResponse
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
    const { container, getByTestId } = render(
      <ArTestWrapper>
        <VersionListPage packageType="DOCKER" />
      </ArTestWrapper>
    )

    const artifact = mockDockerUseGetAllArtifactVersionsQueryResponse.data.content.data
      .artifactVersions?.[0] as ArtifactVersionMetadata

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
    const curlValue = getByText(curlColumn, artifact.pullCommand as string)
    expect(curlValue).toBeInTheDocument()

    const copyCurlBtn = curlColumn.querySelector('[data-icon="duplicate"]') as HTMLElement
    expect(copyCurlBtn).toBeInTheDocument()
    await userEvent.click(copyCurlBtn)
    expect(copy).toHaveBeenCalled()

    const expandIcon = getFirstRowColumn(1).querySelector('[data-icon="chevron-down"') as HTMLElement
    expect(expandIcon).toBeInTheDocument()

    await userEvent.click(rows[0])
    expect(handleToggleExpandableRow).toHaveBeenCalledWith(artifact.name)
  })

  test('Pagination should work', async () => {
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
      return mockEmptyUseGetAllArtifactVersionsQueryResponse
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
        ...mockErrorUseGetAllArtifactVersionsQueryResponse,
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
