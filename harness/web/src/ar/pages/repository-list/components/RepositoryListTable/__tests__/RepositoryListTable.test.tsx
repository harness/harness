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
import { getByTestId, getByText, queryByTestId, render, waitFor } from '@testing-library/react'

import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import repositoryFactory from '@ar/frameworks/RepositoryStep/RepositoryFactory'
import { mockRepositoryListApiResponse } from '@ar/pages/repository-list/__tests__/__mockData__'
import { HelmRepositoryType } from '@ar/pages/repository-details/HelmRepository/HelmRepositoryType'
import { DockerRepositoryType } from '@ar/pages/repository-details/DockerRepository/DockerRepositoryType'
import { getTableColumn, getTableHeaderColumn, getTableRow, testSelectChange } from '@ar/utils/testUtils/utils'
import { GenericRepositoryType } from '@ar/pages/repository-details/GenericRepository/GenericRepositoryType'

import { RepositoryListTable } from '..'

jest.mock('clipboard-copy', () => ({
  __esModule: true,
  default: jest.fn()
}))

const mockHistoryPush = jest.fn()
// eslint-disable-next-line jest-no-mock
jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useHistory: () => ({
    push: mockHistoryPush
  })
}))

describe('Verify Reppository List Table', () => {
  beforeAll(() => {
    repositoryFactory.registerStep(new DockerRepositoryType())
    repositoryFactory.registerStep(new HelmRepositoryType())
    repositoryFactory.registerStep(new GenericRepositoryType())
  })

  beforeEach(() => {
    jest.clearAllMocks()
  })

  test('Verify list should render without any issue', async () => {
    render(
      <ArTestWrapper>
        <RepositoryListTable
          data={mockRepositoryListApiResponse.content.data}
          gotoPage={jest.fn()}
          onPageSizeChange={jest.fn()}
          setSortBy={jest.fn()}
          sortBy={['name', 'DESC']}
        />
      </ArTestWrapper>
    )
    const list = mockRepositoryListApiResponse.content.data.registries
    for (let idx = 0; idx < list.length; idx++) {
      const each = list[idx]
      const row = idx + 1
      // name column
      const nameColumn = getTableColumn(row, 1)
      expect(nameColumn).toHaveTextContent(each.identifier)
      expect(nameColumn?.querySelector('span[data-icon="docker-step"]')).toBeInTheDocument()
      if (each.description) {
        expect(nameColumn?.querySelector('span[data-icon="description"]')).toBeInTheDocument()
      }
      if (each.labels?.length) {
        expect(nameColumn?.querySelector('span[icon="tag"]')).toBeInTheDocument()
      }
      // type column
      expect(getTableColumn(row, 2)).toHaveTextContent(
        each.type === 'VIRTUAL' ? 'badges.artifactRegistry' : 'badges.upstreamProxy'
      )
      // size column
      expect(getTableColumn(row, 3)).toHaveTextContent(each.registrySize ?? 'na')
      // artifacts column
      expect(getTableColumn(row, 4)).toHaveTextContent(each.artifactsCount?.toString() ?? '0')
      // downloads column
      expect(getTableColumn(row, 5)).toHaveTextContent(each.downloadsCount?.toString() ?? '0')
      // copy url column
      const copyUrlBtn = getByText(getTableColumn(row, 7) as HTMLDivElement, 'repositoryList.table.copyUrl')
      await userEvent.click(copyUrlBtn!)
      await waitFor(() => {
        expect(copy).toHaveBeenCalledWith(each.url)
      })
    }
  })

  test('Verify all callbacks working without any issue', async () => {
    const gotoPage = jest.fn()
    const onPageSizeChange = jest.fn()
    const setSortBy = jest.fn()
    const { container } = render(
      <ArTestWrapper>
        <RepositoryListTable
          data={mockRepositoryListApiResponse.content.data}
          gotoPage={gotoPage}
          onPageSizeChange={onPageSizeChange}
          setSortBy={setSortBy}
          sortBy={[]}
        />
      </ArTestWrapper>
    )

    // verify sort
    const nameHeader = getTableHeaderColumn(1)
    await userEvent.click(nameHeader!)
    await waitFor(() => {
      expect(setSortBy).toHaveBeenCalledWith(['identifier', 'DESC'])
    })

    // verify next page change
    const nextPageBtn = container.querySelector('button[aria-label="Next"]')
    await userEvent.click(nextPageBtn!)
    await waitFor(() => {
      expect(gotoPage).toHaveBeenCalledWith(1)
    })

    // verify page size change
    const pageSizeSelector = getByTestId(container, 'dropdown-button')
    expect(pageSizeSelector).toBeInTheDocument()
    await testSelectChange(pageSizeSelector, '10')
    await waitFor(() => {
      expect(onPageSizeChange).toHaveBeenCalledWith('10')
    })

    // verify row click
    const tableRow = getTableRow(1)
    await userEvent.click(tableRow!)
    await waitFor(() => {
      expect(mockHistoryPush).toHaveBeenCalledWith('/registries/repo1')
    })
  })

  test('Verify if no pagination data in response', async () => {
    const { container } = render(
      <ArTestWrapper>
        <RepositoryListTable
          data={{
            ...mockRepositoryListApiResponse.content.data,
            itemCount: undefined,
            pageCount: undefined,
            pageIndex: undefined,
            pageSize: undefined
          }}
          gotoPage={jest.fn()}
          onPageSizeChange={jest.fn()}
          setSortBy={jest.fn()}
          sortBy={[]}
        />
      </ArTestWrapper>
    )

    const nextPageBtn = container.querySelector('button[aria-label="Next"]')
    expect(nextPageBtn).not.toBeInTheDocument()

    const prevPageBtn = container.querySelector('button[aria-label="Prev"]')
    expect(prevPageBtn).not.toBeInTheDocument()

    const pageSizeSelector = queryByTestId(container, 'dropdown-button')
    expect(pageSizeSelector).not.toBeInTheDocument()
  })
})
