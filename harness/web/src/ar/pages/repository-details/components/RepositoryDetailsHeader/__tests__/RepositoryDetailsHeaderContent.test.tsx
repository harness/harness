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
import { getByTestId, render } from '@testing-library/react'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import type { Repository } from '@ar/pages/repository-details/types'
import { MockGetDockerRegistryResponseWithAllData } from '@ar/pages/repository-details/DockerRepository/__tests__/__mockData__'

import { getReadableDateTime } from '@ar/common/dateUtils'
import { DEFAULT_DATE_TIME_FORMAT } from '@ar/constants'

import RepositoryDetailsHeaderContent from '../RepositoryDetailsHeaderContent'
import { MockGetDockerRegistryResponseWithMinimumData } from '../../FormContent/__tests__/__mockData__'
import '../../../RepositoryFactory'

describe('Verify RepositoryDetailsHeaderContent', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  test('Verify with full data', () => {
    const { container } = render(
      <ArTestWrapper>
        <RepositoryDetailsHeaderContent data={MockGetDockerRegistryResponseWithAllData.content.data as Repository} />
      </ArTestWrapper>
    )
    const pageHeader = getByTestId(container, 'registry-header-container')
    expect(pageHeader).toBeInTheDocument()

    expect(pageHeader?.querySelector('span[data-icon=docker-step]')).toBeInTheDocument()
    const data = MockGetDockerRegistryResponseWithAllData.content.data

    const title = getByTestId(container, 'registry-title')
    expect(title).toHaveTextContent(data.identifier)

    const description = getByTestId(container, 'registry-description')
    expect(description).toHaveTextContent(data.description)

    expect(pageHeader?.querySelector('svg[data-icon=tag]')).toBeInTheDocument()

    const lastModifiedAt = getByTestId(container, 'registry-last-modified-at')
    expect(lastModifiedAt).toHaveTextContent(getReadableDateTime(Number(data.modifiedAt), DEFAULT_DATE_TIME_FORMAT))
  })

  test('Verify with minimal data', () => {
    const { container } = render(
      <ArTestWrapper>
        <RepositoryDetailsHeaderContent
          data={MockGetDockerRegistryResponseWithMinimumData.content.data as Repository}
        />
      </ArTestWrapper>
    )
    const pageHeader = getByTestId(container, 'registry-header-container')
    expect(pageHeader).toBeInTheDocument()

    expect(pageHeader?.querySelector('span[data-icon=docker-step]')).toBeInTheDocument()
    const data = MockGetDockerRegistryResponseWithMinimumData.content.data

    const title = getByTestId(container, 'registry-title')
    expect(title).toHaveTextContent(data.identifier)

    const description = getByTestId(container, 'registry-description')
    expect(description).toHaveTextContent('noDescription')

    expect(pageHeader?.querySelector('svg[data-icon=tag]')).not.toBeInTheDocument()

    const lastModifiedAt = getByTestId(container, 'registry-last-modified-at')
    expect(lastModifiedAt).toHaveTextContent('na')
  })
})
