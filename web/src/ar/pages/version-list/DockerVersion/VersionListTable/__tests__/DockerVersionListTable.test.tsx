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
import {
  useGetDockerArtifactManifestsQuery as _useGetDockerArtifactManifestsQuery,
  ListArtifactVersion
} from '@harnessio/react-har-service-client'
import type { ArtifactMetadata, ListArtifact } from '@harnessio/react-har-service-v2-client'
import { getByText, queryByText, render } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import copy from 'clipboard-copy'
import repositoryFactory from '@ar/frameworks/RepositoryStep/RepositoryFactory'
import { handleToggleExpandableRow } from '@ar/components/TableCells/utils'
import { DockerRepositoryType } from '@ar/pages/repository-details/DockerRepository/DockerRepositoryType'
import {
  mockDockerLatestVersionListTableData,
  mockDockerNoPullCmdVersionListTableData,
  mockDockerOldVersionListTableData,
  mockUseGetDockerArtifactManifestsQueryResponse
} from '@ar/pages/version-list/__tests__/__mockData__'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import { getTableColumn } from '@ar/utils/testUtils/utils'
import DockerVersionListTable from '../DockerVersionListTable'

const useGetDockerArtifactManifestsQuery = _useGetDockerArtifactManifestsQuery as jest.Mock

jest.mock('clipboard-copy', () => ({
  __esModule: true,
  default: jest.fn()
}))

jest.mock('@harnessio/react-har-service-client', () => ({
  useGetDockerArtifactManifestsQuery: jest.fn()
}))

jest.mock('@ar/components/TableCells/utils', () => ({
  handleToggleExpandableRow: jest.fn().mockImplementation(() => {
    return (val: Set<any>) => new Set(val)
  })
}))

const convertResponseToV2 = (response: ListArtifactVersion): ListArtifact => {
  return {
    artifacts:
      response.artifactVersions?.map(
        each =>
          ({
            ...each,
            version: each.name,
            package: ''
          } as ArtifactMetadata)
      ) || [],
    ...response
  }
}
describe('Verify Version List Table', () => {
  beforeAll(() => {
    repositoryFactory.registerStep(new DockerRepositoryType())

    useGetDockerArtifactManifestsQuery.mockImplementation(() => {
      return mockUseGetDockerArtifactManifestsQueryResponse
    })
  })

  beforeEach(() => {
    jest.clearAllMocks()
  })

  test('Should render list without any errors', async () => {
    const { container, getByTestId } = render(
      <ArTestWrapper>
        <DockerVersionListTable
          data={convertResponseToV2(mockDockerLatestVersionListTableData)}
          gotoPage={jest.fn()}
          setSortBy={jest.fn()}
          sortBy={['name', 'DESC']}
        />
      </ArTestWrapper>
    )
    const rows = container.querySelectorAll('[class*="TableV2--row"]')
    expect(rows).toHaveLength(1)

    const getFirstRowColumn = (col: number) => getTableColumn(1, col) as HTMLElement

    const artifact = mockDockerLatestVersionListTableData.artifactVersions?.[0]
    const name = getByText(getFirstRowColumn(2), artifact?.name as string)
    expect(name).toBeInTheDocument()

    const nonProdDep = getByTestId('nonProdDeployments')
    const prodDep = getByTestId('prodDeployments')

    expect(getByText(nonProdDep, artifact?.deploymentMetadata?.nonProdEnvCount as number)).toBeInTheDocument()
    expect(getByText(nonProdDep, 'nonProd')).toBeInTheDocument()

    expect(getByText(prodDep, artifact?.deploymentMetadata?.prodEnvCount as number)).toBeInTheDocument()
    expect(getByText(prodDep, 'prod')).toBeInTheDocument()

    const digestValue = getByText(getFirstRowColumn(4), artifact?.digestCount as number)
    expect(digestValue).toBeInTheDocument()

    const curlColumn = getFirstRowColumn(6)
    expect(curlColumn).toHaveTextContent('copy')

    const copyCurlBtn = curlColumn.querySelector('[data-icon="code-copy"]') as HTMLElement
    expect(copyCurlBtn).toBeInTheDocument()
    await userEvent.click(copyCurlBtn)
    expect(copy).toHaveBeenCalled()

    const expandIcon = getFirstRowColumn(1).querySelector('[data-icon="chevron-down"') as HTMLElement
    expect(expandIcon).toBeInTheDocument()

    await userEvent.click(rows[0])
    expect(handleToggleExpandableRow).toHaveBeenCalledWith(artifact?.name)
  })

  test("Should show na if pull command doesn't exist", async () => {
    render(
      <ArTestWrapper>
        <DockerVersionListTable
          data={convertResponseToV2(mockDockerNoPullCmdVersionListTableData)}
          gotoPage={jest.fn()}
          setSortBy={jest.fn()}
          sortBy={['name', 'DESC']}
        />
      </ArTestWrapper>
    )

    const curlColumn = getTableColumn(1, 6) as HTMLElement
    const naCurl = getByText(curlColumn, 'na')
    expect(naCurl).toBeInTheDocument()

    const copyCurlBtn = curlColumn.querySelector('[data-icon="code-copy"]') as HTMLElement
    expect(copyCurlBtn).not.toBeInTheDocument()
  })

  test('Should not show latest tag if item version is not latest', () => {
    render(
      <ArTestWrapper>
        <DockerVersionListTable
          data={convertResponseToV2(mockDockerOldVersionListTableData)}
          gotoPage={jest.fn()}
          setSortBy={jest.fn()}
          sortBy={['version', 'DESC']}
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
        <DockerVersionListTable
          data={null as any}
          gotoPage={jest.fn()}
          setSortBy={jest.fn()}
          sortBy={['name', 'DESC']}
        />
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
        <DockerVersionListTable
          data={convertResponseToV2(mockDockerOldVersionListTableData)}
          gotoPage={jest.fn()}
          setSortBy={setSortBy}
          sortBy={['name', 'DESC']}
        />
      </ArTestWrapper>
    )
    const artifactNameSortIcon = getByText(container, 'versionList.table.columns.version').nextSibling
      ?.firstChild as HTMLElement
    await userEvent.click(artifactNameSortIcon)
    expect(setSortBy).toHaveBeenCalledWith(['version', 'ASC'])
  })
})
