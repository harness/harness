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
import {
  mockUseGetAllArtifactsByRegistryQuery,
  mockEmptyUseGetAllArtifactsByRegistryQuery,
  mockErrorUseGetAllHarnessArtifactsQueryResponse
} from './__mockData__'
import RegistryArtifactListPage from '../RegistryArtifactListPage'

const useGetAllArtifactsByRegistryQuery = _useGetAllArtifactsByRegistryQuery as jest.Mock

jest.mock('@harnessio/react-har-service-client', () => ({
  useGetAllArtifactsByRegistryQuery: jest.fn(),
  useGetArtifactSummaryQuery: jest.fn()
}))

describe('Test Registry Artifact List Page', () => {
  beforeAll(() => {
    useGetAllArtifactsByRegistryQuery.mockImplementation(() => mockUseGetAllArtifactsByRegistryQuery)
  })

  test('Should render empty list if artifacts response is empty', () => {
    useGetAllArtifactsByRegistryQuery.mockImplementationOnce(() => mockEmptyUseGetAllArtifactsByRegistryQuery)
    const { getByText } = render(
      <ArTestWrapper>
        <RegistryArtifactListPage />
      </ArTestWrapper>
    )
    const noResultsText = getByText('artifactList.table.noArtifactsTitle')
    expect(noResultsText).toBeInTheDocument()
  })

  test('Should render artifacts list', () => {
    const { container } = render(
      <ArTestWrapper>
        <RegistryArtifactListPage />
      </ArTestWrapper>
    )
    const table = container.querySelector('[class*="TableV2--table"]')
    expect(table).toBeInTheDocument()

    const tableRows = container.querySelectorAll('[class*="TableV2--row"]')
    expect(tableRows).toHaveLength(1)
  })

  test('Should show error message if listing api fails', async () => {
    const mockRefetchFn = jest.fn().mockImplementation()
    useGetAllArtifactsByRegistryQuery.mockImplementationOnce(() => {
      return {
        ...mockErrorUseGetAllHarnessArtifactsQueryResponse,
        refetch: mockRefetchFn
      }
    })
    const { getByText } = render(
      <ArTestWrapper>
        <RegistryArtifactListPage />
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
    const { getByText, getByPlaceholderText } = render(
      <ArTestWrapper>
        <RegistryArtifactListPage />
      </ArTestWrapper>
    )
    useGetAllArtifactsByRegistryQuery.mockImplementationOnce(() => mockEmptyUseGetAllArtifactsByRegistryQuery)

    const searchInput = getByPlaceholderText('search')
    expect(searchInput).toBeInTheDocument()

    fireEvent.change(searchInput, { target: { value: 'pod' } })
    await waitFor(() =>
      expect(useGetAllArtifactsByRegistryQuery).toHaveBeenLastCalledWith({
        registry_ref: 'undefined/+',
        queryParams: {
          page: 0,
          size: 50,
          search_term: 'pod',
          sort_field: 'updatedAt',
          sort_order: 'DESC'
        },
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      })
    )

    const clearAllFiltersBtn = getByText('clearFilters')
    await userEvent.click(clearAllFiltersBtn)
    expect(useGetAllArtifactsByRegistryQuery).toHaveBeenLastCalledWith({
      registry_ref: 'undefined/+',
      queryParams: {
        page: 0,
        size: 50,
        sort_field: 'updatedAt',
        sort_order: 'DESC'
      },
      stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
    })
  })

  test('Sorting should work', async () => {
    const { getByText } = render(
      <ArTestWrapper>
        <RegistryArtifactListPage />
      </ArTestWrapper>
    )

    const artifactNameSortIcon = getByText('artifactList.table.columns.name').nextSibling?.firstChild as HTMLElement
    await userEvent.click(artifactNameSortIcon)

    expect(useGetAllArtifactsByRegistryQuery).toHaveBeenLastCalledWith({
      registry_ref: 'undefined/+',
      queryParams: {
        page: 0,
        size: 50,
        sort_field: 'name',
        sort_order: 'ASC'
      },
      stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
    })

    const repositorySortIcon = getByText('artifactList.table.columns.repository').nextSibling?.firstChild as HTMLElement
    await userEvent.click(repositorySortIcon)

    expect(useGetAllArtifactsByRegistryQuery).toHaveBeenLastCalledWith({
      registry_ref: 'undefined/+',
      queryParams: {
        page: 0,
        size: 50,
        sort_field: 'registryIdentifier',
        sort_order: 'DESC'
      },
      stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
    })

    const downloadsSortIcon = getByText('artifactList.table.columns.downloads').nextSibling?.firstChild as HTMLElement
    await userEvent.click(downloadsSortIcon)

    expect(useGetAllArtifactsByRegistryQuery).toHaveBeenLastCalledWith({
      registry_ref: 'undefined/+',
      queryParams: {
        page: 0,
        size: 50,
        sort_field: 'downloadsCount',
        sort_order: 'ASC'
      },
      stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
    })

    const lastUpdatedSortIcon = getByText('artifactList.table.columns.latestVersion').nextSibling
      ?.firstChild as HTMLElement
    await userEvent.click(lastUpdatedSortIcon)

    expect(useGetAllArtifactsByRegistryQuery).toHaveBeenLastCalledWith({
      registry_ref: 'undefined/+',
      queryParams: {
        page: 0,
        size: 50,
        sort_field: 'latestVersion',
        sort_order: 'DESC'
      },
      stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
    })
  })

  test('Pagination should work', async () => {
    const { getByText, getByTestId } = render(
      <ArTestWrapper>
        <RegistryArtifactListPage />
      </ArTestWrapper>
    )

    const nextPageBtn = getByText('Next')
    await userEvent.click(nextPageBtn)

    expect(useGetAllArtifactsByRegistryQuery).toHaveBeenLastCalledWith({
      registry_ref: 'undefined/+',
      queryParams: {
        page: 1,
        size: 50,
        sort_field: 'updatedAt',
        sort_order: 'DESC'
      },
      stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
    })

    const pageSizeSelect = getByTestId('dropdown-button')
    await userEvent.click(pageSizeSelect)
    const pageSize20option = getByText('20')
    await userEvent.click(pageSize20option)

    expect(useGetAllArtifactsByRegistryQuery).toHaveBeenLastCalledWith({
      registry_ref: 'undefined/+',
      queryParams: {
        page: 0,
        size: 20,
        sort_field: 'updatedAt',
        sort_order: 'DESC'
      },
      stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
    })
  })
})
