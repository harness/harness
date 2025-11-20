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
import { fireEvent, render, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  useGetAllHarnessArtifactsQuery as _useGetAllHarnessArtifactsQuery,
  useGetAllRegistriesQuery as _useGetAllRegistriesQuery
} from '@harnessio/react-har-service-client'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import { testMultiSelectChange } from '@ar/utils/testUtils/utils'
import repositoryFactory from '@ar/frameworks/RepositoryStep/RepositoryFactory'
import { DockerRepositoryType } from '@ar/pages/repository-details/DockerRepository/DockerRepositoryType'
import { HelmRepositoryType } from '@ar/pages/repository-details/HelmRepository/HelmRepositoryType'
import { GenericRepositoryType } from '@ar/pages/repository-details/GenericRepository/GenericRepositoryType'
import ArtifactListPage from '../ArtifactListPage'
import {
  mockGetAllRegistriesResponse,
  mockEmptyGetAllRegistriesResponse,
  mockEmptyUseGetAllHarnessArtifactsQueryResponse,
  mockUseGetAllHarnessArtifactsQueryResponse,
  mockErrorUseGetAllHarnessArtifactsQueryResponse
} from './__mockData__'

const useGetAllHarnessArtifactsQuery = _useGetAllHarnessArtifactsQuery as jest.Mock
const useGetAllRegistriesQuery = _useGetAllRegistriesQuery as jest.Mock

jest.mock('@harnessio/react-har-service-client', () => ({
  useGetAllHarnessArtifactsQuery: jest.fn(),
  useGetAllRegistriesQuery: jest.fn().mockImplementation(() => ({
    isFetching: false,
    data: mockGetAllRegistriesResponse,
    refetch: jest.fn(),
    error: null
  }))
}))

describe('Test Artifact List Page', () => {
  beforeAll(() => {
    repositoryFactory.registerStep(new DockerRepositoryType())
    repositoryFactory.registerStep(new HelmRepositoryType())
    repositoryFactory.registerStep(new GenericRepositoryType())

    useGetAllHarnessArtifactsQuery.mockImplementation(() => {
      return mockUseGetAllHarnessArtifactsQueryResponse
    })
  })

  test('Should render proper empty list if artifacts reponse is empty', async () => {
    useGetAllHarnessArtifactsQuery.mockImplementationOnce(() => mockEmptyUseGetAllHarnessArtifactsQueryResponse)

    const { container, getByText } = render(
      <ArTestWrapper>
        <ArtifactListPage />
      </ArTestWrapper>
    )
    const noItemsText = getByText('artifactList.table.noArtifactsTitle')
    expect(noItemsText).toBeInTheDocument()

    const noItemsIcon = container.querySelector('[data-icon="store-artifact-bundle"]')
    expect(noItemsIcon).toBeInTheDocument()
  })

  test('Should render artifacts list', () => {
    const { container } = render(
      <ArTestWrapper>
        <ArtifactListPage />
      </ArTestWrapper>
    )

    const table = container.querySelector('[class*="TableV2--table"]')
    expect(table).toBeInTheDocument()

    const rows = container.querySelectorAll('[class*="TableV2--row"]')
    expect(rows).toHaveLength(3)
  })

  test('Should show error message if listing api fails', async () => {
    const mockRefetchFn = jest.fn().mockImplementation(() => undefined)
    useGetAllHarnessArtifactsQuery.mockImplementationOnce(() => {
      return {
        ...mockErrorUseGetAllHarnessArtifactsQueryResponse,
        refetch: mockRefetchFn
      }
    })
    const { getByText } = render(
      <ArTestWrapper>
        <ArtifactListPage />
      </ArTestWrapper>
    )

    const errorText = getByText('error message')
    expect(errorText).toBeInTheDocument()

    const retryBtn = getByText('Retry')
    expect(retryBtn).toBeInTheDocument()

    await userEvent.click(retryBtn)
    expect(mockRefetchFn).toHaveBeenCalled()
  })

  test('Search, Registeries, Package Types and Latest Versions filter should work', async () => {
    const { getByTestId, getByText, getByPlaceholderText } = render(
      <ArTestWrapper>
        <ArtifactListPage />
      </ArTestWrapper>
    )

    expect(useGetAllHarnessArtifactsQuery).toHaveBeenLastCalledWith(
      {
        space_ref: 'undefined/+',
        queryParams: {
          page: 0,
          size: 50,
          sort_field: 'lastModified',
          sort_order: 'DESC',
          reg_identifier: [],
          latest_version: false,
          deployed_artifact: false,
          package_type: []
        },
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      },
      { enabled: true }
    )

    const registriesSelect = getByTestId('regitry-select')
    await testMultiSelectChange(registriesSelect, 'repo1', '')

    expect(useGetAllHarnessArtifactsQuery).toHaveBeenLastCalledWith(
      {
        space_ref: 'undefined/+',
        queryParams: {
          page: 0,
          size: 50,
          sort_field: 'lastModified',
          sort_order: 'DESC',
          reg_identifier: ['repo1'],
          latest_version: false,
          deployed_artifact: false,
          package_type: []
        },
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      },
      { enabled: true }
    )

    const packageTypeSelector = getByTestId('package-type-select')
    expect(packageTypeSelector).toBeInTheDocument()
    await testMultiSelectChange(packageTypeSelector, 'repositoryTypes.docker')

    expect(useGetAllHarnessArtifactsQuery).toHaveBeenLastCalledWith(
      {
        space_ref: 'undefined/+',
        queryParams: {
          page: 0,
          size: 50,
          sort_field: 'lastModified',
          sort_order: 'DESC',
          reg_identifier: ['repo1'],
          latest_version: false,
          deployed_artifact: false,
          package_type: ['DOCKER']
        },
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      },
      { enabled: true }
    )

    const searchInput = getByPlaceholderText('search')
    expect(searchInput).toBeInTheDocument()
    fireEvent.change(searchInput!, { target: { value: '1234' } })
    await waitFor(() =>
      expect(useGetAllHarnessArtifactsQuery).toHaveBeenLastCalledWith(
        {
          space_ref: 'undefined/+',
          queryParams: {
            page: 0,
            size: 50,
            sort_field: 'lastModified',
            sort_order: 'DESC',
            reg_identifier: ['repo1'],
            latest_version: false,
            deployed_artifact: false,
            package_type: ['DOCKER'],
            search_term: '1234'
          },
          stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
        },
        { enabled: true }
      )
    )

    useGetAllHarnessArtifactsQuery.mockImplementationOnce(() => mockEmptyUseGetAllHarnessArtifactsQueryResponse)
    useGetAllRegistriesQuery.mockImplementationOnce(() => mockEmptyGetAllRegistriesResponse)

    const latestVersionTab = getByText('artifactList.table.latestVersions')
    await userEvent.click(latestVersionTab)
    expect(useGetAllHarnessArtifactsQuery).toHaveBeenLastCalledWith(
      {
        space_ref: 'undefined/+',
        queryParams: {
          page: 0,
          size: 50,
          sort_field: 'lastModified',
          sort_order: 'DESC',
          reg_identifier: ['repo1'],
          latest_version: true,
          deployed_artifact: false,
          package_type: ['DOCKER'],
          search_term: '1234'
        },
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      },
      { enabled: true }
    )

    const clearAllFiltersBtn = getByText('clearFilters')
    await userEvent.click(clearAllFiltersBtn)
    expect(useGetAllHarnessArtifactsQuery).toHaveBeenLastCalledWith(
      {
        space_ref: 'undefined/+',
        queryParams: {
          page: 0,
          size: 50,
          sort_field: 'lastModified',
          sort_order: 'DESC',
          reg_identifier: [],
          latest_version: false,
          deployed_artifact: false,
          package_type: []
        },
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      },
      { enabled: true }
    )
  })

  test('Sorting should work', async () => {
    const { getByText } = render(
      <ArTestWrapper>
        <ArtifactListPage />
      </ArTestWrapper>
    )

    const artifactNameSortIcon = getByText('artifactList.table.columns.artifactName').nextSibling
      ?.firstChild as HTMLElement
    await userEvent.click(artifactNameSortIcon)

    expect(useGetAllHarnessArtifactsQuery).toHaveBeenLastCalledWith(
      {
        space_ref: 'undefined/+',
        queryParams: {
          page: 0,
          size: 50,
          sort_field: 'name',
          sort_order: 'ASC',
          reg_identifier: [],
          latest_version: false,
          deployed_artifact: false,
          package_type: []
        },
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      },
      { enabled: true }
    )

    const downloadsSortIcon = getByText('artifactList.table.columns.downloads').nextSibling?.firstChild as HTMLElement
    await userEvent.click(downloadsSortIcon)

    expect(useGetAllHarnessArtifactsQuery).toHaveBeenLastCalledWith(
      {
        space_ref: 'undefined/+',
        queryParams: {
          page: 0,
          size: 50,
          sort_field: 'downloadsCount',
          sort_order: 'DESC',
          reg_identifier: [],
          latest_version: false,
          deployed_artifact: false,
          package_type: []
        },
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      },
      { enabled: true }
    )

    const lastUpdatedSortIcon = getByText('artifactList.table.columns.lastUpdated').nextSibling
      ?.firstChild as HTMLElement
    await userEvent.click(lastUpdatedSortIcon)

    expect(useGetAllHarnessArtifactsQuery).toHaveBeenLastCalledWith(
      {
        space_ref: 'undefined/+',
        queryParams: {
          page: 0,
          size: 50,
          sort_field: 'lastModified',
          sort_order: 'ASC',
          reg_identifier: [],
          latest_version: false,
          deployed_artifact: false,
          package_type: []
        },
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      },
      { enabled: true }
    )
  })

  test('Pagination should work', async () => {
    const { getByText, getByTestId } = render(
      <ArTestWrapper>
        <ArtifactListPage />
      </ArTestWrapper>
    )

    const nextPageBtn = getByText('Next')
    await userEvent.click(nextPageBtn)

    expect(useGetAllHarnessArtifactsQuery).toHaveBeenLastCalledWith(
      {
        space_ref: 'undefined/+',
        queryParams: {
          page: 1,
          size: 50,
          sort_field: 'lastModified',
          sort_order: 'DESC',
          reg_identifier: [],
          latest_version: false,
          deployed_artifact: false,
          package_type: []
        },
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      },
      { enabled: true }
    )

    const pageSizeSelect = getByTestId('dropdown-button')
    await userEvent.click(pageSizeSelect)
    const pageSize20option = getByText('20')
    await userEvent.click(pageSize20option)

    expect(useGetAllHarnessArtifactsQuery).toHaveBeenLastCalledWith(
      {
        space_ref: 'undefined/+',
        queryParams: {
          page: 0,
          size: 20,
          sort_field: 'lastModified',
          sort_order: 'DESC',
          reg_identifier: [],
          latest_version: false,
          deployed_artifact: false,
          package_type: []
        },
        stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
      },
      { enabled: true }
    )
  })
})
