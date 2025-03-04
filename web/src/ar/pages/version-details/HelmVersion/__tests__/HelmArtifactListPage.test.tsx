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
import { fireEvent, getByText as globalGetByText, render, waitFor } from '@testing-library/react'

import '@ar/pages/version-details/VersionFactory'
import '@ar/pages/repository-details/RepositoryFactory'

import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import { RepositoryDetailsTab } from '@ar/pages/repository-details/constants'
import RepositoryDetailsPage from '@ar/pages/repository-details/RepositoryDetailsPage'
import {
  MockGetHelmArtifactsByRegistryResponse,
  MockGetHelmRegistryResponseWithAllData
} from '@ar/pages/repository-details/HelmRepository/__tests__/__mockData__'
import { getTableColumn } from '@ar/utils/testUtils/utils'

const useGetAllArtifactsByRegistryQuery = _useGetAllArtifactsByRegistryQuery as jest.Mock

const deleteArtifact = jest.fn().mockImplementation(() => Promise.resolve({}))

jest.mock('@harnessio/react-har-service-client', () => ({
  useGetAllArtifactsByRegistryQuery: jest.fn(),
  useGetRegistryQuery: jest.fn().mockImplementation(() => ({
    isFetching: false,
    refetch: jest.fn(),
    error: false,
    data: MockGetHelmRegistryResponseWithAllData
  })),
  useDeleteArtifactMutation: jest.fn().mockImplementation(() => ({
    isLoading: false,
    mutateAsync: deleteArtifact
  }))
}))

describe('Test Registry Artifact List Page', () => {
  beforeEach(() => {
    useGetAllArtifactsByRegistryQuery.mockImplementation(() => ({
      isFetching: false,
      data: MockGetHelmArtifactsByRegistryResponse,
      error: false,
      refetch: jest.fn()
    }))
  })

  test('Should render empty list if artifacts response is empty', () => {
    useGetAllArtifactsByRegistryQuery.mockImplementation(() => ({
      isFetching: false,
      data: { content: { data: { artifacts: [] } } },
      error: false,
      refetch: []
    }))
    const { getByText } = render(
      <ArTestWrapper path="/registries/abcd/:tab" pathParams={{ tab: RepositoryDetailsTab.PACKAGES }}>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )
    const noResultsText = getByText('artifactList.table.noArtifactsTitle')
    expect(noResultsText).toBeInTheDocument()
  })

  test('Should render artifacts list', async () => {
    const { container } = render(
      <ArTestWrapper path="/registries/abcd/:tab" pathParams={{ tab: RepositoryDetailsTab.PACKAGES }}>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )
    const table = container.querySelector('[class*="TableV2--table"]')
    expect(table).toBeInTheDocument()

    const tableRows = container.querySelectorAll('[class*="TableV2--row"]')
    expect(tableRows).toHaveLength(1)

    const tableData = MockGetHelmArtifactsByRegistryResponse.content.data.artifacts

    const getFirstRowColumn = (col: number) => getTableColumn(1, col) as HTMLElement
    expect(globalGetByText(getFirstRowColumn(1), tableData[0].name)).toBeInTheDocument()
    expect(globalGetByText(getFirstRowColumn(2), tableData[0].registryIdentifier)).toBeInTheDocument()
    expect(globalGetByText(getFirstRowColumn(3), tableData[0].downloadsCount?.toString() as string)).toBeInTheDocument()
    expect(globalGetByText(getFirstRowColumn(4), tableData[0].latestVersion)).toBeInTheDocument()

    const actionBtn = getFirstRowColumn(5).querySelector('span[data-icon=Options')
    await userEvent.click(actionBtn as HTMLElement)
    const dialogs = document.getElementsByClassName('bp3-popover')
    await waitFor(() => expect(dialogs).toHaveLength(1))
    const selectPopover = dialogs[0] as HTMLElement

    const items = selectPopover.getElementsByClassName('bp3-menu-item')
    expect(items).toHaveLength(2)

    expect(items[0]).toHaveTextContent('artifactList.table.actions.deleteArtifact')
    expect(items[1]).toHaveTextContent('actions.setupClient')
  })

  test('Should show error message if listing api fails', async () => {
    const mockRefetchFn = jest.fn().mockImplementation()
    useGetAllArtifactsByRegistryQuery.mockImplementation(() => ({
      isFetching: false,
      data: null,
      error: {
        message: 'error message'
      },
      refetch: mockRefetchFn
    }))
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
    useGetAllArtifactsByRegistryQuery.mockImplementation(() => ({
      isFetching: false,
      data: { content: { data: { artifacts: [] } } },
      error: false,
      refetch: []
    }))
    const { getByText, getByPlaceholderText } = render(
      <ArTestWrapper path="/registries/abcd/:tab" pathParams={{ tab: RepositoryDetailsTab.PACKAGES }}>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )

    const searchInput = getByPlaceholderText('search')
    expect(searchInput).toBeInTheDocument()

    fireEvent.change(searchInput, { target: { value: 'pod' } })
    await waitFor(() =>
      expect(useGetAllArtifactsByRegistryQuery).toHaveBeenLastCalledWith({
        registry_ref: 'undefined/abcd/+',
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
      registry_ref: 'undefined/abcd/+',
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
      <ArTestWrapper path="/registries/abcd/:tab" pathParams={{ tab: RepositoryDetailsTab.PACKAGES }}>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )

    const artifactNameSortIcon = getByText('artifactList.table.columns.name').nextSibling?.firstChild as HTMLElement
    await userEvent.click(artifactNameSortIcon)

    expect(useGetAllArtifactsByRegistryQuery).toHaveBeenLastCalledWith({
      registry_ref: 'undefined/abcd/+',
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
      registry_ref: 'undefined/abcd/+',
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
      registry_ref: 'undefined/abcd/+',
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
      registry_ref: 'undefined/abcd/+',
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
      <ArTestWrapper path="/registries/abcd/:tab" pathParams={{ tab: RepositoryDetailsTab.PACKAGES }}>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )

    const nextPageBtn = getByText('Next')
    await userEvent.click(nextPageBtn)

    expect(useGetAllArtifactsByRegistryQuery).toHaveBeenLastCalledWith({
      registry_ref: 'undefined/abcd/+',
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
      registry_ref: 'undefined/abcd/+',
      queryParams: {
        page: 0,
        size: 20,
        sort_field: 'updatedAt',
        sort_order: 'DESC'
      },
      stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
    })
  })

  test('Should list actions', async () => {
    render(
      <ArTestWrapper path="/registries/abcd/:tab" pathParams={{ tab: RepositoryDetailsTab.PACKAGES }}>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )

    const getFirstRowColumn = (col: number) => getTableColumn(1, col) as HTMLElement

    // click on 3 dots action btn
    const actionBtn = getFirstRowColumn(5).querySelector('span[data-icon=Options')
    await userEvent.click(actionBtn as HTMLElement)

    const popovers = document.getElementsByClassName('bp3-popover')
    await waitFor(() => expect(popovers).toHaveLength(1))
    const selectPopover = popovers[0] as HTMLElement
    const deleteItem = globalGetByText(selectPopover, 'artifactList.table.actions.deleteArtifact')

    // click on delete action item
    await userEvent.click(deleteItem)
    const dialogs = document.getElementsByClassName('bp3-dialog')
    await waitFor(() => expect(dialogs).toHaveLength(1))
    const deleteDialog = dialogs[0] as HTMLElement
    expect(globalGetByText(deleteDialog, 'artifactDetails.deleteArtifactModal.title')).toBeInTheDocument()
    expect(globalGetByText(deleteDialog, 'artifactDetails.deleteArtifactModal.contentText')).toBeInTheDocument()

    const deleteBtn = deleteDialog.querySelector('button[aria-label=delete]')
    const cancelBtn = deleteDialog.querySelector('button[aria-label=cancel]')
    expect(deleteBtn).toBeInTheDocument()
    expect(cancelBtn).toBeInTheDocument()

    // click on delete button on modal
    await userEvent.click(deleteBtn!)

    await waitFor(() => {
      expect(deleteArtifact).toHaveBeenCalledWith({
        artifact: 'podinfo-artifact/+',
        registry_ref: 'undefined/helm-repo/+'
      })
    })
  })
})
