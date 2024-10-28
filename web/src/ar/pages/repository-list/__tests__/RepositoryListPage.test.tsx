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
import { useGetAllRegistriesQuery } from '@harnessio/react-har-service-client'
import { getByTestId, getByText, render, waitFor } from '@testing-library/react'

import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import { getTableColumn } from '@ar/utils/testUtils/utils'
import repositoryFactory from '@ar/frameworks/RepositoryStep/RepositoryFactory'
import { HelmRepositoryType } from '@ar/pages/repository-details/HelmRepository/HelmRepositoryType'
import { DockerRepositoryType } from '@ar/pages/repository-details/DockerRepository/DockerRepositoryType'
import RepositoryListPage from '../RepositoryListPage'
import { mockRepositoryListApiResponse } from './__mockData__'

jest.mock('clipboard-copy', () => ({
  __esModule: true,
  default: jest.fn()
}))

jest.mock('@harnessio/react-har-service-client', () => ({
  useGetAllRegistriesQuery: jest.fn()
}))

describe('Test Registry List Page', () => {
  beforeAll(() => {
    repositoryFactory.registerStep(new DockerRepositoryType())
    repositoryFactory.registerStep(new HelmRepositoryType())
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

  test('Should show error message if failed to load repo list', () => {
    ;(useGetAllRegistriesQuery as jest.Mock).mockImplementation(() => ({
      isFetching: false,
      data: null,
      refetch: jest.fn(),
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
    expect(container.querySelector('button[aria-label="Retry"]')).toBeInTheDocument()
  })
})
