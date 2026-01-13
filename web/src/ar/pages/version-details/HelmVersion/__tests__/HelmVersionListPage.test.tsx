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
  ArtifactVersionMetadata
} from '@harnessio/react-har-service-client'

import '@ar/pages/version-details/VersionFactory'
import '@ar/pages/repository-details/RepositoryFactory'

import { getTableColumn } from '@ar/utils/testUtils/utils'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import VersionListPage from '@ar/pages/version-list/VersionListPage'

import { mockHelmLatestVersionListTableData } from './__mockData__'

const useGetAllArtifactVersionsQuery = _useGetAllArtifactVersionsQuery as jest.Mock

jest.mock('@harnessio/react-har-service-client', () => ({
  useGetAllArtifactVersionsQuery: jest.fn(),
  useListVersionsQuery: jest.fn()
}))

jest.mock('clipboard-copy', () => ({
  __esModule: true,
  default: jest.fn()
}))

describe('Verify HelmVersion List Page', () => {
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
        <VersionListPage packageType="HELM" />
      </ArTestWrapper>
    )

    const noItemsText = getByTextLocal('versionList.table.noVersionsTitle')
    expect(noItemsText).toBeInTheDocument()
  })

  test('Should render helm version list', async () => {
    useGetAllArtifactVersionsQuery.mockImplementation(() => {
      return {
        data: { content: { data: mockHelmLatestVersionListTableData } },
        loading: false,
        error: null,
        refetch: jest.fn()
      }
    })
    const { container } = render(
      <ArTestWrapper>
        <VersionListPage packageType="HELM" />
      </ArTestWrapper>
    )

    const artifact = mockHelmLatestVersionListTableData.artifactVersions?.[0] as ArtifactVersionMetadata

    const table = container.querySelector('[class*="TableV2--table"]')
    expect(table).toBeInTheDocument()

    const rows = container.querySelectorAll('[class*="TableV2--row"]')
    expect(rows).toHaveLength(1)

    const getFirstRowColumn = (col: number) => getTableColumn(1, col) as HTMLElement

    const name = getByText(getFirstRowColumn(1), artifact.name as string)
    expect(name).toBeInTheDocument()

    const size = getByText(getFirstRowColumn(2), artifact.size as string)
    expect(size).toBeInTheDocument()

    const downloads = getByText(getFirstRowColumn(3), artifact.downloadsCount?.toString() as string)
    expect(downloads).toBeInTheDocument()

    const curlColumn = getFirstRowColumn(5)
    expect(curlColumn).toHaveTextContent('copy')
    const copyCurlBtn = curlColumn.querySelector('[data-icon="code-copy"]') as HTMLElement
    expect(copyCurlBtn).toBeInTheDocument()
    await userEvent.click(copyCurlBtn)
    expect(copy).toHaveBeenCalled()
  })

  test('Pagination should work', async () => {
    useGetAllArtifactVersionsQuery.mockImplementation(() => {
      return {
        data: { content: { data: mockHelmLatestVersionListTableData } },
        loading: false,
        error: null,
        refetch: jest.fn()
      }
    })
    const { getByText: getByTextLocal, getByTestId } = render(
      <ArTestWrapper>
        <VersionListPage packageType="HELM" />
      </ArTestWrapper>
    )

    const nextPageBtn = getByTextLocal('Next')
    await userEvent.click(nextPageBtn)

    expect(useGetAllArtifactVersionsQuery).toHaveBeenLastCalledWith(
      {
        artifact: 'undefined/+',
        queryParams: { page: 1, search_term: '', size: 50, sort_field: 'lastModified', sort_order: 'DESC' },
        registry_ref: 'undefined/+',
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      },
      { enabled: true }
    )

    const pageSizeSelect = getByTestId('dropdown-button')
    await userEvent.click(pageSizeSelect)
    const pageSize20option = getByTextLocal('20')
    await userEvent.click(pageSize20option)

    expect(useGetAllArtifactVersionsQuery).toHaveBeenLastCalledWith(
      {
        artifact: 'undefined/+',
        queryParams: { page: 0, search_term: '', size: 20, sort_field: 'lastModified', sort_order: 'DESC' },
        registry_ref: 'undefined/+',
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      },
      { enabled: true }
    )
  })

  test('Filters should work', async () => {
    useGetAllArtifactVersionsQuery.mockImplementation(() => {
      return {
        data: { content: { data: mockHelmLatestVersionListTableData } },
        loading: false,
        error: null,
        refetch: jest.fn()
      }
    })
    const { getByText: getByTextLocal, getByPlaceholderText } = render(
      <ArTestWrapper>
        <VersionListPage packageType="HELM" />
      </ArTestWrapper>
    )

    expect(useGetAllArtifactVersionsQuery).toHaveBeenLastCalledWith(
      {
        artifact: 'undefined/+',
        queryParams: { page: 0, search_term: '', size: 50, sort_field: 'lastModified', sort_order: 'DESC' },
        registry_ref: 'undefined/+',
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      },
      { enabled: true }
    )

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
      expect(useGetAllArtifactVersionsQuery).toHaveBeenLastCalledWith(
        {
          artifact: 'undefined/+',
          queryParams: { page: 0, search_term: '1234', size: 50, sort_field: 'lastModified', sort_order: 'DESC' },
          registry_ref: 'undefined/+',
          stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
        },
        { enabled: true }
      )
    })

    const clearAllFiltersBtn = getByTextLocal('clearFilters')
    await userEvent.click(clearAllFiltersBtn)

    expect(useGetAllArtifactVersionsQuery).toHaveBeenLastCalledWith(
      {
        artifact: 'undefined/+',
        queryParams: { page: 0, search_term: '', size: 50, sort_field: 'lastModified', sort_order: 'DESC' },
        registry_ref: 'undefined/+',
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      },
      { enabled: true }
    )
  })

  test('Sorting should work', async () => {
    const { getByText: getByTextLocal } = render(
      <ArTestWrapper>
        <VersionListPage packageType="HELM" />
      </ArTestWrapper>
    )

    const artifactNameSortIcon = getByTextLocal('versionList.table.columns.version').nextSibling
      ?.firstChild as HTMLElement
    await userEvent.click(artifactNameSortIcon)
    expect(useGetAllArtifactVersionsQuery).toHaveBeenLastCalledWith(
      {
        artifact: 'undefined/+',
        queryParams: { page: 0, search_term: '', size: 50, sort_field: 'name', sort_order: 'ASC' },
        registry_ref: 'undefined/+',
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      },
      { enabled: true }
    )
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
        <VersionListPage packageType="HELM" />
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
