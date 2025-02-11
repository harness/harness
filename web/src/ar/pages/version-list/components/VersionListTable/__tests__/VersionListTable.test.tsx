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
import { getByText, queryByText, render } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import copy from 'clipboard-copy'
import repositoryFactory from '@ar/frameworks/RepositoryStep/RepositoryFactory'
import {
  mockHelmLatestVersionListTableData,
  mockHelmNoPullCmdVersionListTableData,
  mockHelmOldVersionListTableData
} from '@ar/pages/version-list/__tests__/__mockData__'
import { HelmRepositoryType } from '@ar/pages/repository-details/HelmRepository/HelmRepositoryType'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import { getTableColumn } from '@ar/utils/testUtils/utils'
import VersionListTable from '../VersionListTable'

jest.mock('clipboard-copy', () => ({
  __esModule: true,
  default: jest.fn()
}))

describe('Verify Version List Table', () => {
  beforeAll(() => {
    repositoryFactory.registerStep(new HelmRepositoryType())
  })

  beforeEach(() => {
    jest.clearAllMocks()
  })

  test('Should render list without any errors', async () => {
    const { container } = render(
      <ArTestWrapper>
        <VersionListTable
          data={mockHelmLatestVersionListTableData}
          gotoPage={jest.fn()}
          setSortBy={jest.fn()}
          sortBy={['name', 'DESC']}
        />
      </ArTestWrapper>
    )
    const rows = container.querySelectorAll('[class*="TableV2--row"]')
    expect(rows).toHaveLength(1)

    const getFirstRowColumn = (col: number) => getTableColumn(1, col) as HTMLElement
    const latestTag = getByText(getFirstRowColumn(1), 'tags.latest')
    expect(latestTag).toBeInTheDocument()
    const sizeValue = getByText(
      getFirstRowColumn(2),
      mockHelmLatestVersionListTableData.artifactVersions?.[0].size as string
    )
    expect(sizeValue).toBeInTheDocument()
    const downloadsValue = getByText(
      getFirstRowColumn(3),
      mockHelmLatestVersionListTableData.artifactVersions?.[0].downloadsCount as number
    )
    expect(downloadsValue).toBeInTheDocument()
    const curlColumn = getFirstRowColumn(5)
    expect(curlColumn).toHaveTextContent('copy')
    const copyCurlBtn = curlColumn.querySelector('[data-icon="code-copy"]') as HTMLElement
    expect(copyCurlBtn).toBeInTheDocument()
    await userEvent.click(copyCurlBtn)
    expect(copy).toHaveBeenCalled()
  })

  test("Should show na if pull command doesn't exist", async () => {
    render(
      <ArTestWrapper>
        <VersionListTable
          data={mockHelmNoPullCmdVersionListTableData}
          gotoPage={jest.fn()}
          setSortBy={jest.fn()}
          sortBy={['name', 'DESC']}
        />
      </ArTestWrapper>
    )

    const curlColumn = getTableColumn(1, 5) as HTMLElement
    const naCurl = getByText(curlColumn, 'na')
    expect(naCurl).toBeInTheDocument()

    const copyCurlBtn = curlColumn.querySelector('[data-icon="duplicate"]') as HTMLElement
    expect(copyCurlBtn).not.toBeInTheDocument()
  })

  test('Should not show latest tag if item version is not latest', () => {
    render(
      <ArTestWrapper>
        <VersionListTable
          data={mockHelmOldVersionListTableData}
          gotoPage={jest.fn()}
          setSortBy={jest.fn()}
          sortBy={['name', 'DESC']}
        />
      </ArTestWrapper>
    )
    const getFirstTableColumn = (col: number) => getTableColumn(col, 1) as HTMLElement
    const latestTag = queryByText(getFirstTableColumn(1), 'tags.latest')
    expect(latestTag).not.toBeInTheDocument()
  })

  test('Should show no rows if no data is provided', () => {
    const { container } = render(
      <ArTestWrapper>
        <VersionListTable data={null as any} gotoPage={jest.fn()} setSortBy={jest.fn()} sortBy={['name', 'DESC']} />
      </ArTestWrapper>
    )
    const table = container.querySelector('[class*="TableV2--table"]')
    expect(table).toBeInTheDocument()

    const rows = container.querySelector('[class*="TableV2--row"]')
    expect(rows).not.toBeInTheDocument()
  })

  test('Should be able to sort', async () => {
    const setSortBy = jest.fn()

    const { container } = render(
      <ArTestWrapper>
        <VersionListTable
          data={mockHelmOldVersionListTableData}
          gotoPage={jest.fn()}
          setSortBy={setSortBy}
          sortBy={['name', 'DESC']}
        />
      </ArTestWrapper>
    )
    const artifactNameSortIcon = getByText(container, 'versionList.table.columns.version').nextSibling
      ?.firstChild as HTMLElement
    await userEvent.click(artifactNameSortIcon)
    expect(setSortBy).toHaveBeenCalledWith(['name', 'ASC'])
  })
})
