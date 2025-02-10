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
import { render } from '@testing-library/react'

import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'

import RepositoryDetailsPage from '../RepositoryDetailsPage'
import { MockGetDockerRegistryResponseWithAllData } from '../DockerRepository/__tests__/__mockData__'

jest.mock('@harnessio/react-har-service-client', () => ({
  useGetRegistryQuery: jest.fn().mockImplementation(() => ({
    isFetching: false,
    refetch: jest.fn(),
    error: false,
    data: MockGetDockerRegistryResponseWithAllData
  }))
}))

describe('Verify Remaining Scenarios for RepositoryDetailsPage', () => {
  test('Verify if selected tab is not in the list of tabs', () => {
    const { container } = render(
      <ArTestWrapper path="/registries/abcd/:tab" pathParams={{ tab: 'abcd' }}>
        <RepositoryDetailsPage />
      </ArTestWrapper>
    )
    expect(container).toHaveTextContent('stepNotFound')
  })
})
