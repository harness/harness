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
import { queryByTestId, render } from '@testing-library/react'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'

import { Parent } from '@ar/common/types'
import type { VirtualRegistryRequest } from '@ar/pages/repository-details/types'

import { queryByNameAttribute } from 'utils/test/testUtils'

import { RepositoryFormComponent } from './TestFormUtils'
import RepositoryConfigurationFormContent from '../RepositoryConfigurationFormContent'
import {
  MockGetDockerRegistryResponseWithMinimumData,
  MockGetDockerRegistryResponseWithMinimumDataForOSS
} from './__mockData__'

import '../../../RepositoryFactory'

jest.mock('@harnessio/react-har-service-client', () => ({
  useGetAllRegistriesQuery: jest.fn().mockImplementation(() => ({
    isFetching: false,
    data: { content: { data: { registries: [] }, status: 'SUCCESS' } },
    refetch: jest.fn(),
    error: null
  }))
}))

describe('Verify repository configuration form content', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  test('Verify with minimal data', () => {
    const { container } = render(
      <ArTestWrapper parent={Parent.Enterprise}>
        <RepositoryFormComponent
          initialValues={MockGetDockerRegistryResponseWithMinimumData.content.data as VirtualRegistryRequest}
          onSubmit={jest.fn()}>
          <RepositoryConfigurationFormContent readonly={false} />
        </RepositoryFormComponent>
      </ArTestWrapper>
    )

    const nameField = queryByNameAttribute('identifier', container)
    expect(nameField).toBeInTheDocument()
    expect(nameField).toHaveAttribute('value', MockGetDockerRegistryResponseWithMinimumData.content.data.identifier)

    // Security scan section
    const securityScanSection = queryByTestId(container, 'security-scan-section')
    expect(securityScanSection).toBeInTheDocument()

    // upstream proxy section
    const upstreamProxySection = queryByTestId(container, 'upstream-proxy-section')
    expect(upstreamProxySection).not.toBeInTheDocument()

    // artifact filtering rules
    const filteringRulesSection = queryByTestId(container, 'include-exclude-patterns-section')
    expect(filteringRulesSection).not.toBeInTheDocument()

    // cleanup policy section
    const cleanupPoliciesSection = queryByTestId(container, 'cleanup-policy-section')
    expect(cleanupPoliciesSection).not.toBeInTheDocument()
  })

  test('Verify with minimal data for OSS', () => {
    const { container } = render(
      <ArTestWrapper parent={Parent.OSS}>
        <RepositoryFormComponent
          initialValues={MockGetDockerRegistryResponseWithMinimumDataForOSS.content.data as VirtualRegistryRequest}
          onSubmit={jest.fn()}>
          <RepositoryConfigurationFormContent readonly={false} />
        </RepositoryFormComponent>
      </ArTestWrapper>
    )

    const nameField = queryByNameAttribute('identifier', container)
    expect(nameField).toBeInTheDocument()
    expect(nameField).toHaveAttribute(
      'value',
      MockGetDockerRegistryResponseWithMinimumDataForOSS.content.data.identifier
    )

    // Security scan section
    const securityScanSection = queryByTestId(container, 'security-scan-section')
    expect(securityScanSection).not.toBeInTheDocument()

    // upstream proxy section
    const upstreamProxySection = queryByTestId(container, 'upstream-proxy-section')
    expect(upstreamProxySection).toBeInTheDocument()

    // artifact filtering rules
    const filteringRulesSection = queryByTestId(container, 'include-exclude-patterns-section')
    expect(filteringRulesSection).not.toBeInTheDocument()

    // cleanup policy section
    const cleanupPoliciesSection = queryByTestId(container, 'cleanup-policy-section')
    expect(cleanupPoliciesSection).not.toBeInTheDocument()
  })
})
