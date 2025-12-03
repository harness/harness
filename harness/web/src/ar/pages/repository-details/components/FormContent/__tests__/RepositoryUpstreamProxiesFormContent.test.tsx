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
import { getByTestId, getByText, render } from '@testing-library/react'
import { useGetAllRegistriesQuery } from '@harnessio/react-har-service-client'

import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import type { VirtualRegistryRequest } from '@ar/pages/repository-details/types'
import { MockGetDockerRegistryResponseWithAllData } from '@ar/pages/repository-details/DockerRepository/__tests__/__mockData__'

import { RepositoryFormComponent } from './TestFormUtils'
import RepositoryUpstreamProxiesFormContent from '../RepositoryUpstreamProxiesFormContent'
import { MockGetDockerRegistryResponseWithMinimumData, MockGetUpstreamProxyRegistryListResponse } from './__mockData__'

jest.mock('@harnessio/react-har-service-client', () => ({
  useGetAllRegistriesQuery: jest.fn().mockImplementation(() => ({
    isFetching: false,
    data: MockGetUpstreamProxyRegistryListResponse,
    refetch: jest.fn(),
    error: null
  }))
}))

describe('Verify Upstream Proxies select', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  test('Verify flow with initial value not empty', async () => {
    const { container } = render(
      <ArTestWrapper>
        <RepositoryFormComponent
          initialValues={MockGetDockerRegistryResponseWithAllData.content.data as VirtualRegistryRequest}
          onSubmit={jest.fn()}>
          <RepositoryUpstreamProxiesFormContent isEdit={false} disabled={false} />
        </RepositoryFormComponent>
      </ArTestWrapper>
    )

    expect(
      container.querySelector('button[aria-label="repositoryDetails.upstreamProxiesSelectList.addUpstreamProxies"]')
    ).not.toBeInTheDocument()

    const selectedItems = container.querySelectorAll('ul[aria-label=orderable-list] .bp3-menu-item')
    selectedItems.forEach((each, idx) => {
      expect(each).toHaveTextContent(MockGetDockerRegistryResponseWithAllData.content.data.config.upstreamProxies[idx])
    })

    const selectableItems = container.querySelectorAll('ul[aria-label=selectable-list] .bp3-menu-item')
    selectableItems.forEach(async (each, idx) => {
      expect(each).toHaveTextContent(MockGetUpstreamProxyRegistryListResponse.content.data.registries[idx].identifier)
    })
  })

  test('Verify flow with initial value empty', async () => {
    ;(useGetAllRegistriesQuery as jest.Mock).mockImplementation(() => ({
      isFetching: false,
      data: { content: { data: { registries: [] }, status: 'SUCCESS' } },
      refetch: jest.fn(),
      error: null
    }))
    const { container } = render(
      <ArTestWrapper>
        <RepositoryFormComponent
          initialValues={MockGetDockerRegistryResponseWithMinimumData.content.data as VirtualRegistryRequest}
          onSubmit={jest.fn()}>
          <RepositoryUpstreamProxiesFormContent isEdit={false} disabled={false} />
        </RepositoryFormComponent>
      </ArTestWrapper>
    )

    const section = getByTestId(container, 'upstream-proxy-section')
    expect(section).toBeInTheDocument()
    expect(getByText(container, 'repositoryDetails.repositoryForm.upstreamProxiesTitle')).toBeInTheDocument()
    expect(getByText(container, 'repositoryDetails.repositoryForm.upstreamProxiesSubTitle')).toBeInTheDocument()

    const configureUpstreamBtn = container.querySelector(
      'button[aria-label="repositoryDetails.upstreamProxiesSelectList.addUpstreamProxies"]'
    )
    expect(configureUpstreamBtn).toBeInTheDocument()
    await userEvent.click(configureUpstreamBtn!)

    const selectableList = container.querySelector('ul[aria-label=selectable-list]') as HTMLElement
    expect(selectableList).toBeInTheDocument()
    expect(getByText(selectableList, 'No data')).toBeInTheDocument()

    expect(container.querySelector('ul[aria-label=orderable-list]')).toBeInTheDocument()
  })

  test('Verify loading api flow', async () => {
    ;(useGetAllRegistriesQuery as jest.Mock).mockImplementation(() => ({
      isFetching: true,
      data: null,
      refetch: jest.fn(),
      error: null
    }))
    const { container } = render(
      <ArTestWrapper>
        <RepositoryFormComponent
          initialValues={MockGetDockerRegistryResponseWithAllData.content.data as VirtualRegistryRequest}
          onSubmit={jest.fn()}>
          <RepositoryUpstreamProxiesFormContent isEdit={false} disabled={false} />
        </RepositoryFormComponent>
      </ArTestWrapper>
    )

    const selectableList = container.querySelector('ul[aria-label=selectable-list]') as HTMLElement
    expect(selectableList).toBeInTheDocument()
    expect(getByText(selectableList, 'Loading...')).toBeInTheDocument()
  })

  test('Verify error api flow', async () => {
    ;(useGetAllRegistriesQuery as jest.Mock).mockImplementation(() => ({
      isFetching: false,
      data: null,
      refetch: jest.fn(),
      error: { message: 'failed to load data' }
    }))
    const { container } = render(
      <ArTestWrapper>
        <RepositoryFormComponent
          initialValues={MockGetDockerRegistryResponseWithAllData.content.data as VirtualRegistryRequest}
          onSubmit={jest.fn()}>
          <RepositoryUpstreamProxiesFormContent isEdit={false} disabled={false} />
        </RepositoryFormComponent>
      </ArTestWrapper>
    )

    const selectableList = container.querySelector('ul[aria-label=selectable-list]') as HTMLElement
    expect(selectableList).toBeInTheDocument()
    expect(getByText(selectableList, 'failed to load data')).toBeInTheDocument()
  })
})
