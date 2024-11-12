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
import type { FormikProps } from 'formik'
import { useToaster } from '@harnessio/uicore'
import userEvent from '@testing-library/user-event'
import { fireEvent, render, waitFor } from '@testing-library/react'
import { useCreateRegistryMutation } from '@harnessio/react-har-service-client'

import { RepositoryPackageType } from '@ar/common/types'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import repositoryFactory from '@ar/frameworks/RepositoryStep/RepositoryFactory'
import { HelmRepositoryType } from '@ar/pages/repository-details/HelmRepository/HelmRepositoryType'
import { DockerRepositoryType } from '@ar/pages/repository-details/DockerRepository/DockerRepositoryType'

import { queryByNameAttribute } from 'utils/test/testUtils'
import RepositoryCreateForm from '../RepositoryCreateForm'

const mutateCreateRegistrySuccess = jest.fn().mockImplementation(
  () =>
    new Promise(onSuccess => {
      onSuccess({
        content: {
          status: 'SUCCESS',
          data: { identifier: '1234' }
        }
      })
    })
)

const mutateCreateRegistryFailure = jest.fn().mockImplementation(
  () =>
    new Promise((_, onFailed) => {
      onFailed({
        message: 'Custom message for failed scenario'
      })
    })
)

jest.mock('@harnessio/react-har-service-client', () => ({
  useCreateRegistryMutation: jest.fn()
}))

jest.mock('@harnessio/uicore', () => ({
  ...jest.requireActual('@harnessio/uicore'),
  useToaster: jest.fn().mockImplementation(() => ({
    showSuccess: jest.fn,
    showError: jest.fn(),
    clear: jest.fn()
  }))
}))

describe('Verify RepositoryCreateForm', () => {
  beforeAll(() => {
    repositoryFactory.registerStep(new DockerRepositoryType())
    repositoryFactory.registerStep(new HelmRepositoryType())
  })

  beforeEach(() => {
    jest.clearAllMocks()
  })

  test('Should render form without error and should submit form without error', async () => {
    ;(useCreateRegistryMutation as jest.Mock).mockImplementation(() => ({
      isLoading: false,
      mutateAsync: mutateCreateRegistrySuccess
    }))

    const setShowOverlay = jest.fn()
    const onSuccess = jest.fn()
    const formikRef = React.createRef<FormikProps<unknown>>()

    const { container } = render(
      <ArTestWrapper>
        <RepositoryCreateForm setShowOverlay={setShowOverlay} onSuccess={onSuccess} ref={formikRef} />
      </ArTestWrapper>
    )

    const selectPackageTypeLabel = 'repositoryDetails.repositoryForm.selectRepoType'
    expect(container).toHaveTextContent(selectPackageTypeLabel)

    const dockerType = container.querySelector('input[type="checkbox"][name="packageType"][value="DOCKER"]')
    expect(dockerType).toBeChecked()

    const helmType = container.querySelector('input[type="checkbox"][name="packageType"][value="HELM"]')
    expect(helmType).not.toBeChecked()
    await userEvent.click(container.querySelector('span[data-icon="service-helm"]')!)
    await waitFor(() => {
      expect(helmType).toBeChecked()
    })

    const nameField = queryByNameAttribute('identifier', container)
    expect(nameField).toBeInTheDocument()
    fireEvent.change(nameField!, { target: { value: 'helm-repo-1' } })

    formikRef.current?.submitForm()
    await waitFor(() => {
      expect(mutateCreateRegistrySuccess).toHaveBeenCalledWith({
        body: {
          cleanupPolicy: [],
          config: { type: 'VIRTUAL', upstreamProxies: [] },
          identifier: 'helm-repo-1',
          packageType: 'HELM',
          parentRef: 'undefined',
          scanners: []
        },
        queryParams: { space_ref: 'undefined' }
      })
    })
    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalledWith({ identifier: '1234' })
    })
  })

  test('Should work form validations correctly', async () => {
    ;(useCreateRegistryMutation as jest.Mock).mockImplementation(() => ({
      isLoading: false,
      mutateAsync: mutateCreateRegistrySuccess
    }))

    const setShowOverlay = jest.fn()
    const onSuccess = jest.fn()
    const formikRef = React.createRef<FormikProps<unknown>>()

    const { container } = render(
      <ArTestWrapper>
        <RepositoryCreateForm setShowOverlay={setShowOverlay} onSuccess={onSuccess} ref={formikRef} />
      </ArTestWrapper>
    )

    formikRef.current?.submitForm()
    await waitFor(() => {
      expect(mutateCreateRegistrySuccess).not.toHaveBeenCalled()
    })
    await waitFor(() => {
      expect(onSuccess).not.toHaveBeenCalled()
    })

    const nameField = queryByNameAttribute('identifier', container)
    expect(nameField).toBeInTheDocument()
    fireEvent.change(nameField!, { target: { value: 'repo-1' } })

    formikRef.current?.submitForm()
    await waitFor(() => {
      expect(mutateCreateRegistrySuccess).toHaveBeenCalled()
    })
    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalled()
    })
  })

  test('Should select default type based on defaultType prop', () => {
    const formikRef = React.createRef<FormikProps<unknown>>()

    const { container } = render(
      <ArTestWrapper>
        <RepositoryCreateForm
          defaultType={RepositoryPackageType.HELM}
          setShowOverlay={jest.fn}
          onSuccess={jest.fn}
          ref={formikRef}
        />
      </ArTestWrapper>
    )

    const dockerType = container.querySelector('input[type="checkbox"][name="packageType"][value="DOCKER"]')
    expect(dockerType).not.toBeChecked()

    const helmType = container.querySelector('input[type="checkbox"][name="packageType"][value="HELM"]')
    expect(helmType).toBeChecked()
  })

  test('Should disable types based on allowedTypes prop', () => {
    const formikRef = React.createRef<FormikProps<unknown>>()

    const { container } = render(
      <ArTestWrapper>
        <RepositoryCreateForm
          allowedPackageTypes={[RepositoryPackageType.HELM]}
          setShowOverlay={jest.fn}
          onSuccess={jest.fn}
          ref={formikRef}
        />
      </ArTestWrapper>
    )

    const dockerType = container.querySelector('input[type="checkbox"][name="packageType"][value="DOCKER"]')
    expect(dockerType).toBeDisabled()

    const helmType = container.querySelector('input[type="checkbox"][name="packageType"][value="HELM"]')
    expect(helmType).not.toBeDisabled()
    expect(helmType).toBeChecked()
  })

  test('Should show error if failed to create registry', async () => {
    ;(useCreateRegistryMutation as jest.Mock).mockImplementation(() => ({
      isLoading: false,
      mutateAsync: mutateCreateRegistryFailure
    }))

    const showError = jest.fn()
    ;(useToaster as jest.Mock).mockImplementation(() => ({
      showSuccess: jest.fn,
      showError,
      clear: jest.fn()
    }))

    const setShowOverlay = jest.fn()
    const onSuccess = jest.fn()
    const formikRef = React.createRef<FormikProps<unknown>>()

    const { container } = render(
      <ArTestWrapper>
        <RepositoryCreateForm setShowOverlay={setShowOverlay} onSuccess={onSuccess} ref={formikRef} />
      </ArTestWrapper>
    )

    const nameField = queryByNameAttribute('identifier', container)
    expect(nameField).toBeInTheDocument()
    fireEvent.change(nameField!, { target: { value: 'docker-repo-1' } })

    formikRef.current?.submitForm()
    await waitFor(() => {
      expect(mutateCreateRegistryFailure).toHaveBeenCalledWith({
        body: {
          cleanupPolicy: [],
          config: { type: 'VIRTUAL', upstreamProxies: [] },
          identifier: 'docker-repo-1',
          packageType: 'DOCKER',
          parentRef: 'undefined',
          scanners: []
        },
        queryParams: { space_ref: 'undefined' }
      })
    })
    await waitFor(() => {
      expect(showError).toHaveBeenCalledWith('Custom message for failed scenario')
    })
  })

  test('Should render default text if passed incorrect type or disabled type to defaultType', () => {
    const formikRef = React.createRef<FormikProps<unknown>>()

    const { container } = render(
      <ArTestWrapper>
        <RepositoryCreateForm
          defaultType={'RANDOM' as RepositoryPackageType}
          setShowOverlay={jest.fn}
          onSuccess={jest.fn}
          ref={formikRef}
        />
      </ArTestWrapper>
    )
    const nameField = queryByNameAttribute('identifier', container)
    expect(nameField).not.toBeInTheDocument()
  })
})
