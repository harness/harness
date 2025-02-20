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
import { queryByText, render } from '@testing-library/react'

import { RepositoryPackageType } from '@ar/common/types'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import type { VirtualRegistryRequest } from '@ar/pages/repository-details/types'
import repositoryFactory from '@ar/frameworks/RepositoryStep/RepositoryFactory'

import { RepositoryFormComponent } from './TestFormUtils'
import { MockGetDockerRegistryResponseWithMinimumData } from './__mockData__'
import SelectContainerScannersFormSection from '../SelectContainerScannersFormSection'

import '../../../RepositoryFactory'

describe('verify SelectContainerScannersFormSection', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  test('Verify when scanners are available', () => {
    const { container } = render(
      <ArTestWrapper>
        <RepositoryFormComponent
          initialValues={MockGetDockerRegistryResponseWithMinimumData.content.data as VirtualRegistryRequest}
          onSubmit={jest.fn()}>
          <SelectContainerScannersFormSection packageType={RepositoryPackageType.DOCKER} />
        </RepositoryFormComponent>
      </ArTestWrapper>
    )

    const checkboxes = container.querySelectorAll('label.bp3-control.bp3-checkbox')
    const supportedScanners =
      repositoryFactory.getRepositoryType(RepositoryPackageType.DOCKER)?.getSupportedScanners() || []

    checkboxes.forEach((each, idx) => {
      const ele = each.querySelector(`input[value=${supportedScanners[idx]}][type=checkbox]`)
      expect(ele).toBeInTheDocument()
      expect(ele).toBeChecked()
      expect(ele).toBeDisabled()
    })
  })

  test('Verify when scanners are not available', () => {
    const { container } = render(
      <ArTestWrapper>
        <RepositoryFormComponent
          initialValues={MockGetDockerRegistryResponseWithMinimumData.content.data as VirtualRegistryRequest}
          onSubmit={jest.fn()}>
          <SelectContainerScannersFormSection packageType={RepositoryPackageType.HELM} />
        </RepositoryFormComponent>
      </ArTestWrapper>
    )

    expect(queryByText(container, 'repositoryDetails.repositoryForm.securityScan.title')).not.toBeInTheDocument()
    const checkboxes = container.querySelector('label.bp3-control.bp3-checkbox')
    expect(checkboxes).not.toBeInTheDocument()
  })

  test('Verify when invalid package type', () => {
    const { container } = render(
      <ArTestWrapper>
        <RepositoryFormComponent
          initialValues={MockGetDockerRegistryResponseWithMinimumData.content.data as VirtualRegistryRequest}
          onSubmit={jest.fn()}>
          <SelectContainerScannersFormSection packageType={'DUMMY' as RepositoryPackageType} />
        </RepositoryFormComponent>
      </ArTestWrapper>
    )

    expect(queryByText(container, 'repositoryDetails.repositoryForm.securityScan.title')).not.toBeInTheDocument()
    const checkboxes = container.querySelector('label.bp3-control.bp3-checkbox')
    expect(checkboxes).not.toBeInTheDocument()
  })
})
