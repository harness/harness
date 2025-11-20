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
import { useGetAllRegistriesQuery } from '@harnessio/react-har-service-client'
import { fireEvent, getByPlaceholderText, getByTestId, render, waitFor } from '@testing-library/react'
import { GenericRepositoryType } from '@ar/pages/repository-details/GenericRepository/GenericRepositoryType'

import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import { getTableHeaderColumn, testMultiSelectChange, testSelectChange } from '@ar/utils/testUtils/utils'
import repositoryFactory from '@ar/frameworks/RepositoryStep/RepositoryFactory'
import { HelmRepositoryType } from '@ar/pages/repository-details/HelmRepository/HelmRepositoryType'
import { DockerRepositoryType } from '@ar/pages/repository-details/DockerRepository/DockerRepositoryType'
import RepositoryListPage from '../RepositoryListPage'
import { mockRepositoryListApiResponse } from './__mockData__'

jest.mock('@harnessio/react-har-service-client', () => ({
  useGetAllRegistriesQuery: jest.fn()
}))

describe('Test Registry List Page', () => {
  beforeAll(() => {
    repositoryFactory.registerStep(new DockerRepositoryType())
    repositoryFactory.registerStep(new HelmRepositoryType())
    repositoryFactory.registerStep(new GenericRepositoryType())
  })

  beforeEach(() => {
    jest.clearAllMocks()
  })

  test('Should render empty list view', async () => {
    ;(useGetAllRegistriesQuery as jest.Mock).mockImplementation(() => ({
      isFetching: false,
      data: { content: { data: { registries: [] }, status: 'SUCCESS' } },
      refetch: jest.fn(),
      error: null
    }))

    const { container } = render(
      <ArTestWrapper>
        <RepositoryListPage />
      </ArTestWrapper>
    )
    expect(container.querySelector('span[data-icon="registry"]')).toBeInTheDocument()
    expect(container).toHaveTextContent('repositoryList.table.noRepositoriesTitle')
    expect(
      container.querySelector('.NoDataCard--buttonContainer--niZYe button[aria-label="repositoryList.newRepository"]')
    ).toBeInTheDocument()
  })

  test('Should render list without any error', async () => {
    ;(useGetAllRegistriesQuery as jest.Mock).mockImplementation(() => ({
      isFetching: false,
      data: mockRepositoryListApiResponse,
      refetch: jest.fn(),
      error: false
    }))
    const { container } = render(
      <ArTestWrapper>
        <RepositoryListPage />
      </ArTestWrapper>
    )
    expect(getByTestId(container, 'page-header')).toHaveTextContent('repositoryList.pageHeading')
    expect(container.querySelector('button[aria-label="repositoryList.newRepository"]')).toHaveTextContent(
      'repositoryList.newRepository'
    )
    expect(getByTestId(container, 'package-type-select')).toBeInTheDocument()
    expect(container.querySelector('.ui-search-box')).toBeInTheDocument()

    const table = document.querySelector('div[class*="TableV2--table--"]')
    expect(table).toBeInTheDocument()
  })

  test('Should show error message if failed to load repo list', async () => {
    const refetch = jest.fn()
    ;(useGetAllRegistriesQuery as jest.Mock).mockImplementation(() => ({
      isFetching: false,
      data: null,
      refetch,
      error: {
        message: 'Failed to load with custom error message'
      }
    }))

    const { container } = render(
      <ArTestWrapper>
        <RepositoryListPage />
      </ArTestWrapper>
    )
    expect(container.querySelector('span[icon="error"]')).toBeInTheDocument()
    expect(container).toHaveTextContent('Failed to load with custom error message')
    const retryBtn = container.querySelector('button[aria-label="Retry"]')
    expect(retryBtn).toBeInTheDocument()

    await userEvent.click(retryBtn!)
    await waitFor(() => {
      expect(refetch).toHaveBeenCalled()
    })
  })

  test('Should show empty message if not results after applying filters', async () => {
    ;(useGetAllRegistriesQuery as jest.Mock).mockImplementation(() => ({
      isFetching: false,
      data: { content: { data: { registries: [] }, status: 'SUCCESS' } },
      refetch: jest.fn(),
      error: null
    }))

    const { container } = render(
      <ArTestWrapper queryParams={{ searchTerm: '123' }}>
        <RepositoryListPage />
      </ArTestWrapper>
    )
    expect(container.querySelector('span[data-icon="registry"]')).toBeInTheDocument()
    expect(container).toHaveTextContent('noResultsFound')
    const clearFilterBtn = container.querySelector('button[aria-label="clearFilters"]')
    expect(clearFilterBtn).toBeInTheDocument()
    await userEvent.click(clearFilterBtn!)
    await waitFor(() => {
      expect(useGetAllRegistriesQuery).toHaveBeenLastCalledWith(
        {
          queryParams: {
            package_type: [],
            page: 0,
            size: 50,
            sort_field: 'lastModified',
            sort_order: 'DESC',
            scope: 'none'
          },
          space_ref: 'undefined/+',
          stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
        },
        { enabled: true }
      )
    })
  })

  test('Should call api after modifying filters on table', async () => {
    ;(useGetAllRegistriesQuery as jest.Mock).mockImplementation(() => ({
      isFetching: false,
      data: mockRepositoryListApiResponse,
      refetch: jest.fn(),
      error: false
    }))

    const { container } = render(
      <ArTestWrapper>
        <RepositoryListPage />
      </ArTestWrapper>
    )

    await waitFor(() => {
      expect(useGetAllRegistriesQuery).toHaveBeenLastCalledWith(
        {
          queryParams: {
            package_type: [],
            page: 0,
            size: 50,
            sort_field: 'lastModified',
            sort_order: 'DESC',
            scope: 'none'
          },
          space_ref: 'undefined/+',
          stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
        },
        { enabled: true }
      )
    })

    const registryTypeSelector = getByTestId(container, 'registry-type-select')
    expect(registryTypeSelector).toBeInTheDocument()
    await testSelectChange(registryTypeSelector, 'repositoryList.artifactRegistry.label')
    await waitFor(() => {
      expect(useGetAllRegistriesQuery).toHaveBeenLastCalledWith(
        {
          queryParams: {
            package_type: [],
            page: 0,
            size: 50,
            sort_field: 'lastModified',
            sort_order: 'DESC',
            type: 'VIRTUAL',
            scope: 'none'
          },
          space_ref: 'undefined/+',
          stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
        },
        { enabled: true }
      )
    })

    const packageTypeSelector = getByTestId(container, 'package-type-select')
    expect(packageTypeSelector).toBeInTheDocument()
    await testMultiSelectChange(packageTypeSelector, 'repositoryTypes.docker')
    await waitFor(() => {
      expect(useGetAllRegistriesQuery).toHaveBeenLastCalledWith(
        {
          queryParams: {
            package_type: ['DOCKER'],
            page: 0,
            size: 50,
            sort_field: 'lastModified',
            sort_order: 'DESC',
            type: 'VIRTUAL',
            scope: 'none'
          },
          space_ref: 'undefined/+',
          stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
        },
        { enabled: true }
      )
    })

    const searchInput = getByPlaceholderText(container, 'search')
    expect(searchInput).toBeInTheDocument()
    fireEvent.change(searchInput!, { target: { value: '1234' } })

    await waitFor(() => {
      expect(useGetAllRegistriesQuery).toHaveBeenLastCalledWith(
        {
          queryParams: {
            package_type: ['DOCKER'],
            page: 0,
            size: 50,
            sort_field: 'lastModified',
            sort_order: 'DESC',
            type: 'VIRTUAL',
            search_term: '1234',
            scope: 'none'
          },
          space_ref: 'undefined/+',
          stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
        },
        { enabled: true }
      )
    })

    // should call an api onchange on sort
    const nameHeader = getTableHeaderColumn(1)
    await userEvent.click(nameHeader!)
    await waitFor(() => {
      expect(useGetAllRegistriesQuery).toHaveBeenLastCalledWith(
        {
          queryParams: {
            package_type: ['DOCKER'],
            page: 0,
            size: 50,
            sort_field: 'identifier',
            sort_order: 'ASC',
            type: 'VIRTUAL',
            search_term: '1234',
            scope: 'none'
          },
          space_ref: 'undefined/+',
          stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
        },
        { enabled: true }
      )
    })

    // should call api on page change
    const nextPageBtn = container.querySelector('button[aria-label="Next"]')
    await userEvent.click(nextPageBtn!)
    await waitFor(() => {
      expect(useGetAllRegistriesQuery).toHaveBeenLastCalledWith(
        {
          queryParams: {
            package_type: ['DOCKER'],
            page: 1,
            size: 50,
            sort_field: 'identifier',
            sort_order: 'ASC',
            type: 'VIRTUAL',
            search_term: '1234',
            scope: 'none'
          },
          space_ref: 'undefined/+',
          stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
        },
        { enabled: true }
      )
    })

    //should call api on page size change
    const pageSizeSelector = getByTestId(container, 'dropdown-button')
    expect(pageSizeSelector).toBeInTheDocument()
    await testSelectChange(pageSizeSelector, '10')
    await waitFor(() => {
      expect(useGetAllRegistriesQuery).toHaveBeenLastCalledWith(
        {
          queryParams: {
            package_type: ['DOCKER'],
            page: 0,
            size: 10,
            sort_field: 'identifier',
            sort_order: 'ASC',
            type: 'VIRTUAL',
            search_term: '1234',
            scope: 'none'
          },
          space_ref: 'undefined/+',
          stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
        },
        { enabled: true }
      )
    })
  })
})
