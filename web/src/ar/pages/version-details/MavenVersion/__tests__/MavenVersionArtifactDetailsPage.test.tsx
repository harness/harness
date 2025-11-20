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
import { getByText, render } from '@testing-library/react'
import { useGetArtifactFilesQuery } from '@harnessio/react-har-service-client'

import { getTableColumn } from '@ar/utils/testUtils/utils'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'

import VersionDetailsPage from '../../VersionDetailsPage'
import { mockMavenArtifactFiles, mockMavenVersionList, mockMavenVersionSummary } from './__mockData__'

const mockHistoryPush = jest.fn()
// eslint-disable-next-line jest-no-mock
jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useHistory: () => ({
    push: mockHistoryPush
  })
}))

jest.mock('clipboard-copy', () => ({
  __esModule: true,
  default: jest.fn()
}))

jest.mock('@harnessio/react-har-service-client', () => ({
  useGetArtifactVersionSummaryQuery: jest.fn().mockImplementation(() => ({
    data: { content: mockMavenVersionSummary },
    error: null,
    isLoading: false,
    refetch: jest.fn()
  })),
  getAllArtifactVersions: jest.fn().mockImplementation(
    () =>
      new Promise(success => {
        success({ content: mockMavenVersionList })
      })
  ),
  useGetArtifactFilesQuery: jest.fn().mockImplementation(() => ({
    data: { content: mockMavenArtifactFiles },
    error: null,
    isLoading: false,
    refetch: jest.fn()
  }))
}))

describe('Verify Maven Artifact Version Artifact Details Tab', () => {
  test('verify file list table', async () => {
    const { container } = render(
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/artifacts/:artifactIdentifier/versions/:versionIdentifier/:versionTab"
        pathParams={{
          repositoryIdentifier: '1',
          artifactIdentifier: '1',
          versionIdentifier: '1',
          versionTab: 'artifact_details'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    expect(container).toBeInTheDocument()

    const getTableRowColumn = (row: number, col: number) => getTableColumn(row, col) as HTMLElement

    const data = mockMavenArtifactFiles.data.files[0]
    const row = 1
    expect(getTableRowColumn(row, 1)).toHaveTextContent(data.name)
    expect(getTableRowColumn(row, 2)).toHaveTextContent(data?.size as string)

    const column3Content = getTableRowColumn(row, 3)
    data.checksums.forEach(async each => {
      const [checksum, value] = each.split(': ')
      const btn = getByText(column3Content, `copy ${checksum}`)
      await userEvent.click(btn)
      expect(copy).toHaveBeenCalledWith(value)
    })

    const copyColumn = getTableRowColumn(row, 4)
    expect(copyColumn).toHaveTextContent('copy')
    const copyBtn = copyColumn.querySelector('[data-icon="code-copy"]') as HTMLElement
    expect(copyBtn).toBeInTheDocument()
    await userEvent.click(copyBtn)
    expect(copy).toHaveBeenCalledWith(data?.downloadCommand)
  })

  test('verify pagination and sorting actions', async () => {
    const { container } = render(
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/artifacts/:artifactIdentifier/versions/:versionIdentifier/:versionTab"
        pathParams={{
          repositoryIdentifier: '1',
          artifactIdentifier: '1',
          versionIdentifier: '1',
          versionTab: 'artifact_details'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )
    const nextPageBtn = getByText(container, 'Next')
    await userEvent.click(nextPageBtn)

    expect(useGetArtifactFilesQuery).toHaveBeenLastCalledWith({
      artifact: '1/+',
      queryParams: { page: 0, size: 50, sort_field: 'createdAt', sort_order: 'DESC' },
      registry_ref: 'undefined/1/+',
      version: '1'
    })

    const fileNameSortIcon = getByText(container, 'versionDetails.artifactFiles.table.columns.name').nextSibling
      ?.firstChild as HTMLElement
    await userEvent.click(fileNameSortIcon)

    expect(useGetArtifactFilesQuery).toHaveBeenLastCalledWith({
      artifact: '1/+',
      queryParams: { page: 0, size: 50, sort_field: 'createdAt', sort_order: 'DESC' },
      registry_ref: 'undefined/1/+',
      version: '1'
    })
  })

  test('verify error message', async () => {
    const mockRefetchFn = jest.fn().mockImplementation(() => undefined)
    ;(useGetArtifactFilesQuery as jest.Mock).mockImplementationOnce(() => {
      return {
        data: null,
        loading: false,
        error: { message: 'error message' },
        refetch: mockRefetchFn
      }
    })

    const { container } = render(
      <ArTestWrapper
        path="/registries/:repositoryIdentifier/artifacts/:artifactIdentifier/versions/:versionIdentifier/:versionTab"
        pathParams={{
          repositoryIdentifier: '1',
          artifactIdentifier: '1',
          versionIdentifier: '1',
          versionTab: 'artifact_details'
        }}>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    const errorText = getByText(container, 'error message')
    expect(errorText).toBeInTheDocument()

    const retryBtn = getByText(container, 'Retry')
    expect(retryBtn).toBeInTheDocument()

    await userEvent.click(retryBtn)
    expect(mockRefetchFn).toHaveBeenCalled()
  })
})
