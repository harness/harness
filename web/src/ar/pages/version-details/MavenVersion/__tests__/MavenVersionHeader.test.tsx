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
import { getByTestId, render, waitFor } from '@testing-library/react'
import { getAllArtifactVersions } from '@harnessio/react-har-service-client'

import '@ar/pages/version-details/VersionFactory'
import '@ar/pages/repository-details/RepositoryFactory'
import { testSelectChange } from '@ar/utils/testUtils/utils'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'

import VersionDetailsPage from '../../VersionDetailsPage'
import { mockMavenVersionList, mockMavenVersionSummary } from './__mockData__'

const mockHistoryPush = jest.fn()
// eslint-disable-next-line jest-no-mock
jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useHistory: () => ({
    push: mockHistoryPush
  })
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
  )
}))

describe('Verify MavenVersionHeader component render', () => {
  test('Verify breadcrumbs', () => {
    const { container } = render(
      <ArTestWrapper>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    const pageHeader = getByTestId(container, 'page-header')
    expect(pageHeader).toBeInTheDocument()

    const breadcrumbsSection = pageHeader?.querySelector('div[class*=PageHeader--breadcrumbsDiv--]')
    expect(breadcrumbsSection).toBeInTheDocument()

    expect(breadcrumbsSection).toHaveTextContent('breadcrumbs.repositories: undefined')
    expect(breadcrumbsSection).toHaveTextContent('breadcrumbs.artifacts: undefined')
  })

  test('Verify icon, name, selectors and other actions', () => {
    const { container } = render(
      <ArTestWrapper>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    const pageHeader = getByTestId(container, 'page-header')
    expect(pageHeader).toBeInTheDocument()

    expect(pageHeader?.querySelector('span[data-icon=maven-repository-type]')).toBeInTheDocument()

    const data = mockMavenVersionSummary.data
    expect(pageHeader).toHaveTextContent(data.imageName)

    const versionSelector = getByTestId(container, 'version-select')
    expect(versionSelector).toBeInTheDocument()

    const setupClientBtn = pageHeader.querySelector('button[aria-label="actions.setupClient"]')
    expect(setupClientBtn).toBeInTheDocument()
  })

  test('verify version selector: Success Case', async () => {
    const { container } = render(
      <ArTestWrapper>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    const versionSelector = getByTestId(container, 'version-select')
    expect(versionSelector).toBeInTheDocument()

    const data = mockMavenVersionSummary.data
    expect(versionSelector).toHaveTextContent(data.version)

    await userEvent.click(versionSelector)
    await testSelectChange(versionSelector, '1.0.1', data.version)

    await waitFor(() => {
      expect(mockHistoryPush).toHaveBeenLastCalledWith('/registries/artifacts/versions/1.0.1')
    })
  })

  test('verify version selector: Failure Case', async () => {
    ;(getAllArtifactVersions as jest.Mock).mockImplementationOnce(
      () =>
        new Promise((_, failure) => {
          failure({ message: 'Failed to fetch versions' })
        })
    )
    const { container } = render(
      <ArTestWrapper>
        <VersionDetailsPage />
      </ArTestWrapper>
    )

    const versionSelector = getByTestId(container, 'version-select')
    expect(versionSelector).toBeInTheDocument()

    const data = mockMavenVersionSummary.data
    expect(versionSelector).toHaveTextContent(data.version)

    await userEvent.click(versionSelector)
    const dialogs = document.getElementsByClassName('bp3-select-popover')
    await waitFor(() => expect(dialogs).toHaveLength(1))
    const selectPopover = dialogs[0] as HTMLElement
    expect(selectPopover).toHaveTextContent('No items found')
  })
})
