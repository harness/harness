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
  type ArtifactVersionMetadata
} from '@harnessio/react-har-service-client'

import userEvent from '@testing-library/user-event'
import repositoryFactory from '@ar/frameworks/RepositoryStep/RepositoryFactory'
import { HelmRepositoryType } from '@ar/pages/repository-details/HelmRepository/HelmRepositoryType'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import { HelmVersionType } from '@ar/pages/version-details/HelmVersion/HelmVersionType'
import versionFactory from '@ar/frameworks/Version/VersionFactory'
import { getTableColumn } from '@ar/utils/testUtils/utils'
import VersionListPage from '../VersionListPage'
import {
  mockEmptyUseGetAllArtifactVersionsQueryResponse,
  mockErrorUseGetAllArtifactVersionsQueryResponse,
  mockHelmOldVersionListTableData,
  mockHelmUseGetAllArtifactVersionsQueryResponse
} from './__mockData__'

const useGetAllArtifactVersionsQuery = _useGetAllArtifactVersionsQuery as jest.Mock

jest.mock('@harnessio/react-har-service-client', () => ({
  useGetAllArtifactVersionsQuery: jest.fn(),
  useListVersionsQuery: jest.fn()
}))

jest.mock('clipboard-copy', () => ({
  __esModule: true,
  default: jest.fn()
}))

describe('Verify Version List Page', () => {
  beforeAll(() => {
    repositoryFactory.registerStep(new HelmRepositoryType())
    versionFactory.registerStep(new HelmVersionType())

    useGetAllArtifactVersionsQuery.mockImplementation(() => {
      return mockHelmUseGetAllArtifactVersionsQueryResponse
    })
  })

  test('Should render empty list text if response is empty', () => {
    useGetAllArtifactVersionsQuery.mockImplementationOnce(() => {
      return mockEmptyUseGetAllArtifactVersionsQueryResponse
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
    const { container } = render(
      <ArTestWrapper>
        <VersionListPage packageType="HELM" />
      </ArTestWrapper>
    )

    const table = container.querySelector('[class*="TableV2--table"]')
    expect(table).toBeInTheDocument()

    const rows = container.querySelectorAll('[class*="TableV2--row"]')
    expect(rows).toHaveLength(1)

    const artifact = mockHelmOldVersionListTableData.artifactVersions?.[0] as ArtifactVersionMetadata

    const getFirstRowColumn = (col: number) => getTableColumn(1, col) as HTMLElement
    const name = getByText(getFirstRowColumn(1), artifact.name)
    expect(name).toBeInTheDocument()

    const sizeValue = getByText(getFirstRowColumn(2), artifact.size as string)
    expect(sizeValue).toBeInTheDocument()

    const downloadsValue = getByText(getFirstRowColumn(3), artifact.downloadsCount as number)
    expect(downloadsValue).toBeInTheDocument()

    const curlColumn = getFirstRowColumn(5)
    expect(curlColumn).toHaveTextContent('copy')
    const copyCurlBtn = curlColumn.querySelector('[data-icon="code-copy"]') as HTMLElement
    expect(copyCurlBtn).toBeInTheDocument()
    await userEvent.click(copyCurlBtn)
    expect(copy).toHaveBeenCalled()
  })

  test('Pagination should work', async () => {
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

  test('Filter should work', async () => {
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
      return mockEmptyUseGetAllArtifactVersionsQueryResponse
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
        ...mockErrorUseGetAllArtifactVersionsQueryResponse,
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
