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
import { useGetAllArtifactsByRegistryQuery as _useGetAllArtifactsByRegistryQuery } from '@harnessio/react-har-service-client'
import { fireEvent, render, waitFor } from '@testing-library/react'

import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import { RepositoryDetailsTab } from '@ar/pages/repository-details/constants'
import RepositoryDetailsPage from '@ar/pages/repository-details/RepositoryDetailsPage'
import { MockGetDockerRegistryResponseWithAllData } from '@ar/pages/repository-details/DockerRepository/__tests__/__mockData__'

import {
  mockUseGetAllArtifactsByRegistryQuery,
  mockEmptyUseGetAllArtifactsByRegistryQuery,
  mockErrorUseGetAllHarnessArtifactsQueryResponse
} from './__mockData__'

const useGetAllArtifactsByRegistryQuery = _useGetAllArtifactsByRegistryQuery as jest.Mock

jest.mock('@harnessio/react-har-service-client', () => ({
  useListPackagesQuery: jest.fn(),
  useGetAllArtifactsByRegistryQuery: jest.fn(),
  useGetArtifactSummaryQuery: jest.fn(),
  useGetRegistryQuery: jest.fn().mockImplementation(() => ({
    isFetching: false,
    refetch: jest.fn(),
    error: false,
    data: MockGetDockerRegistryResponseWithAllData
  }))
}))

describe('Test Registry Artifact List Page', () => {
  beforeEach(() => {
    useGetAllArtifactsByRegistryQuery.mockImplementation(() => mockUseGetAllArtifactsByRegistryQuery)
  })

  test('Should render empty list if artifacts response is empty', () => {
    useGetAllArtifactsByRegistryQuery.mockImplementation(() => mockEmptyUseGetAllArtifactsByRegistryQuery)
    const { getByText } = render(
      <ArTestWrapper path="/registries/abcd/:tab" pathParams={{ tab: RepositoryDetailsTab.PACKAGES }}>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )
    const noResultsText = getByText('artifactList.table.noArtifactsTitle')
    expect(noResultsText).toBeInTheDocument()
  })

  test('Should render artifacts list', () => {
    const { container } = render(
      <ArTestWrapper path="/registries/abcd/:tab" pathParams={{ tab: RepositoryDetailsTab.PACKAGES }}>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )
    const table = container.querySelector('[class*="TableV2--table"]')
    expect(table).toBeInTheDocument()

    const tableRows = container.querySelectorAll('[class*="TableV2--row"]')
    expect(tableRows).toHaveLength(1)
  })

  test('Should show error message if listing api fails', async () => {
    const mockRefetchFn = jest.fn().mockImplementation()
    useGetAllArtifactsByRegistryQuery.mockImplementation(() => {
      return {
        ...mockErrorUseGetAllHarnessArtifactsQueryResponse,
        refetch: mockRefetchFn
      }
    })
    const { getByText } = render(
      <ArTestWrapper path="/registries/abcd/:tab" pathParams={{ tab: RepositoryDetailsTab.PACKAGES }}>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )

    const errorText = getByText('error message')
    expect(errorText).toBeInTheDocument()

    const retryBtn = getByText('Retry')
    expect(retryBtn).toBeInTheDocument()
    await userEvent.click(retryBtn)
    expect(mockRefetchFn).toHaveBeenCalled()
  })

  test('Should be able to search', async () => {
    useGetAllArtifactsByRegistryQuery.mockImplementation(() => mockEmptyUseGetAllArtifactsByRegistryQuery)
    const { getByText, getByPlaceholderText } = render(
      <ArTestWrapper path="/registries/abcd/:tab" pathParams={{ tab: RepositoryDetailsTab.PACKAGES }}>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )

    const searchInput = getByPlaceholderText('search')
    expect(searchInput).toBeInTheDocument()

    fireEvent.change(searchInput, { target: { value: 'pod' } })
    await waitFor(() =>
      expect(useGetAllArtifactsByRegistryQuery).toHaveBeenLastCalledWith(
        {
          registry_ref: 'undefined/abcd/+',
          queryParams: {
            page: 0,
            size: 50,
            search_term: 'pod',
            sort_field: 'lastModified',
            sort_order: 'DESC'
          },
          stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
        },
        { enabled: true }
      )
    )

    const clearAllFiltersBtn = getByText('clearFilters')
    await userEvent.click(clearAllFiltersBtn)
    expect(useGetAllArtifactsByRegistryQuery).toHaveBeenLastCalledWith(
      {
        registry_ref: 'undefined/abcd/+',
        queryParams: {
          page: 0,
          size: 50,
          sort_field: 'lastModified',
          sort_order: 'DESC'
        },
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      },
      { enabled: true }
    )
  })

  test('Sorting should work', async () => {
    const { getByText } = render(
      <ArTestWrapper path="/registries/abcd/:tab" pathParams={{ tab: RepositoryDetailsTab.PACKAGES }}>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )

    const artifactNameSortIcon = getByText('artifactList.table.columns.name').nextSibling?.firstChild as HTMLElement
    await userEvent.click(artifactNameSortIcon)

    expect(useGetAllArtifactsByRegistryQuery).toHaveBeenLastCalledWith(
      {
        registry_ref: 'undefined/abcd/+',
        queryParams: {
          page: 0,
          size: 50,
          sort_field: 'name',
          sort_order: 'ASC'
        },
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      },
      { enabled: true }
    )

    const repositorySortIcon = getByText('artifactList.table.columns.repository').nextSibling?.firstChild as HTMLElement
    await userEvent.click(repositorySortIcon)

    expect(useGetAllArtifactsByRegistryQuery).toHaveBeenLastCalledWith(
      {
        registry_ref: 'undefined/abcd/+',
        queryParams: {
          page: 0,
          size: 50,
          sort_field: 'registryIdentifier',
          sort_order: 'DESC'
        },
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      },
      { enabled: true }
    )

    const downloadsSortIcon = getByText('artifactList.table.columns.downloads').nextSibling?.firstChild as HTMLElement
    await userEvent.click(downloadsSortIcon)

    expect(useGetAllArtifactsByRegistryQuery).toHaveBeenLastCalledWith(
      {
        registry_ref: 'undefined/abcd/+',
        queryParams: {
          page: 0,
          size: 50,
          sort_field: 'downloadsCount',
          sort_order: 'ASC'
        },
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      },
      { enabled: true }
    )

    const lastUpdatedSortIcon = getByText('artifactList.table.columns.latestVersion').nextSibling
      ?.firstChild as HTMLElement
    await userEvent.click(lastUpdatedSortIcon)

    expect(useGetAllArtifactsByRegistryQuery).toHaveBeenLastCalledWith(
      {
        registry_ref: 'undefined/abcd/+',
        queryParams: {
          page: 0,
          size: 50,
          sort_field: 'lastModified',
          sort_order: 'DESC'
        },
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      },
      { enabled: true }
    )
  })

  test('Pagination should work', async () => {
    const { getByText, getByTestId } = render(
      <ArTestWrapper path="/registries/abcd/:tab" pathParams={{ tab: RepositoryDetailsTab.PACKAGES }}>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )

    const nextPageBtn = getByText('Next')
    await userEvent.click(nextPageBtn)

    expect(useGetAllArtifactsByRegistryQuery).toHaveBeenLastCalledWith(
      {
        registry_ref: 'undefined/abcd/+',
        queryParams: {
          page: 1,
          size: 50,
          sort_field: 'lastModified',
          sort_order: 'DESC'
        },
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      },
      { enabled: true }
    )

    const pageSizeSelect = getByTestId('dropdown-button')
    await userEvent.click(pageSizeSelect)
    const pageSize20option = getByText('20')
    await userEvent.click(pageSize20option)

    expect(useGetAllArtifactsByRegistryQuery).toHaveBeenLastCalledWith(
      {
        registry_ref: 'undefined/abcd/+',
        queryParams: {
          page: 0,
          size: 20,
          sort_field: 'lastModified',
          sort_order: 'DESC'
        },
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      },
      { enabled: true }
    )
  })
})
