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
import { useGetArtifactSummaryQuery } from '@harnessio/react-har-service-client'
import { getByTestId, queryByTestId, render, waitFor } from '@testing-library/react'

import '@ar/pages/version-details/VersionFactory'
import '@ar/pages/repository-details/RepositoryFactory'
import { DEFAULT_DATE_TIME_FORMAT } from '@ar/constants'
import { getReadableDateTime } from '@ar/common/dateUtils'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'

import ArtifactDetailsPage from '../ArtifactDetailsPage'
import { MOCK_GENERIC_ARTIFACT_SUMMARY } from './__mockData__'

jest.mock('@harnessio/react-har-service-client', () => ({
  useGetArtifactSummaryQuery: jest.fn().mockImplementation(() => {
    return {
      data: MOCK_GENERIC_ARTIFACT_SUMMARY,
      loading: false,
      error: null,
      refetch: jest.fn()
    }
  }),
  useGetAllArtifactVersionsQuery: jest.fn().mockImplementation(() => {
    return {
      data: { content: { data: [] } },
      error: null,
      loading: false,
      refetch: jest.fn()
    }
  })
}))

describe('Verify Artifact Details Page', () => {
  test('Should render Header without any error', () => {
    const { container } = render(
      <ArTestWrapper>
        <ArtifactDetailsPage />
      </ArTestWrapper>
    )

    const pageHeader = getByTestId(container, 'page-header')
    expect(pageHeader).toBeInTheDocument()

    const data = MOCK_GENERIC_ARTIFACT_SUMMARY.content.data
    expect(pageHeader).toHaveTextContent(data.imageName)
    expect(pageHeader?.querySelector('span[data-icon=generic-repository-type]')).toBeInTheDocument()
    expect(pageHeader).toHaveTextContent('artifactDetails.totalDownloads' + data.downloadsCount)
    expect(pageHeader).toHaveTextContent(getReadableDateTime(Number(data.createdAt), DEFAULT_DATE_TIME_FORMAT))
    expect(pageHeader).toHaveTextContent(getReadableDateTime(Number(data.modifiedAt), DEFAULT_DATE_TIME_FORMAT))
  })

  test('Should show error message with retry button if failed to load summary api', async () => {
    const retryFn = jest.fn()
    ;(useGetArtifactSummaryQuery as jest.Mock).mockImplementation(() => {
      return {
        data: null,
        loading: false,
        error: { message: 'Failed to load with custom error message' },
        refetch: retryFn
      }
    })
    const { container } = render(
      <ArTestWrapper>
        <ArtifactDetailsPage />
      </ArTestWrapper>
    )

    expect(container.querySelector('span[icon="error"]')).toBeInTheDocument()
    expect(container).toHaveTextContent('Failed to load with custom error message')
    const retryBtn = container.querySelector('button[aria-label="Retry"]')
    expect(retryBtn).toBeInTheDocument()

    await userEvent.click(retryBtn!)
    await waitFor(() => {
      expect(retryFn).toHaveBeenCalled()
    })
  })

  test('Should not render header if no data or empty data', async () => {
    const retryFn = jest.fn()
    ;(useGetArtifactSummaryQuery as jest.Mock).mockImplementation(() => {
      return {
        data: {},
        loading: false,
        error: null,
        refetch: retryFn
      }
    })
    const { container } = render(
      <ArTestWrapper>
        <ArtifactDetailsPage />
      </ArTestWrapper>
    )

    const pageHeader = queryByTestId(container, 'page-header')
    expect(pageHeader).not.toBeInTheDocument()
  })

  test('Should show loading for api loading', async () => {
    const retryFn = jest.fn()
    ;(useGetArtifactSummaryQuery as jest.Mock).mockImplementation(() => {
      return {
        data: null,
        isFetching: true,
        error: null,
        refetch: retryFn
      }
    })
    const { container } = render(
      <ArTestWrapper>
        <ArtifactDetailsPage />
      </ArTestWrapper>
    )

    const pageSpinner = queryByTestId(container, 'page-spinner')
    expect(pageSpinner).toBeInTheDocument()
  })
})
