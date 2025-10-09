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
import { getByPlaceholderText, getByTestId, getByText, render } from '@testing-library/react'

import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import RepositoryDetailsPage from '@ar/pages/repository-details/RepositoryDetailsPage'
import { RepositoryDetailsTab } from '@ar/pages/repository-details/constants'
import { MockGetDockerRegistryResponseWithAllData } from '@ar/pages/repository-details/DockerRepository/__tests__/__mockData__'

jest.mock('@harnessio/react-har-service-client', () => ({
  useGetRegistryQuery: jest.fn().mockImplementation(() => ({
    isFetching: false,
    refetch: jest.fn(),
    error: false,
    data: MockGetDockerRegistryResponseWithAllData
  })),
  useListWebhooksQuery: jest.fn().mockImplementation(() => ({
    isFetching: false,
    refetch: jest.fn(),
    error: false,
    data: { content: { data: { webhooks: [] }, status: 'SUCCESS' } }
  }))
}))

describe('Test Registry Webhook List Page', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  test('Should render empty list if webhooks response is empty', () => {
    const { container } = render(
      <ArTestWrapper
        featureFlags={{
          HAR_TRIGGERS: true
        }}
        path="/registries/abcd/:tab"
        pathParams={{ tab: RepositoryDetailsTab.WEBHOOKS }}>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )

    expect(container.querySelector('[data-icon="code-webhook"]')).toBeInTheDocument()
    expect(container).toHaveTextContent('webhookList.table.noWebhooksTitle')

    const pageSubHeader = getByTestId(container, 'page-subheader')
    const createWebhookButton = getByText(pageSubHeader, 'webhookList.newWebhook')
    expect(createWebhookButton).toBeInTheDocument()

    const searchInput = getByPlaceholderText(pageSubHeader, 'search')
    expect(searchInput).toBeInTheDocument()
  })
})
