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
import { Button } from '@harnessio/uicore'
import userEvent from '@testing-library/user-event'
import { fireEvent, render, waitFor } from '@testing-library/react'

import { RepositoryPackageType } from '@ar/common/types'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import repositoryFactory from '@ar/frameworks/RepositoryStep/RepositoryFactory'
import { CreateRepository } from '@ar/pages/repository-list/components/CreateRepository/CreateRepository'

import { queryByNameAttribute } from 'utils/test/testUtils'
import { HelmRepositoryType } from '../../HelmRepository/HelmRepositoryType'
import { DockerRepositoryType } from '../../DockerRepository/DockerRepositoryType'
import {
  useCreateRepositoryModal,
  useCreateRepositoryModalProps
} from '../useCreateRepositoryModal/useCreateRepositoryModal'

const mutateCreateRegistry = jest.fn().mockImplementation(
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

jest.mock('@harnessio/react-har-service-client', () => ({
  useCreateRegistryMutation: jest.fn().mockImplementation(() => ({
    isLoading: false,
    mutateAsync: mutateCreateRegistry
  }))
}))

const openModalDialog = async (container: HTMLElement): Promise<HTMLElement> => {
  const mainBtn = container.querySelector('button[aria-label="repositoryList.newRepository"]')
  expect(mainBtn!).toBeInTheDocument()
  await userEvent.click(mainBtn!)

  const modalDialog = document.querySelector('div.bp3-dialog') as HTMLElement
  await waitFor(() => {
    expect(modalDialog).toBeInTheDocument()
  })
  return modalDialog
}

const CustomCreateRepositoryModal = (props: useCreateRepositoryModalProps) => {
  const [open] = useCreateRepositoryModal(props)
  return <Button text="repositoryList.newRepository" onClick={open} />
}

describe('Verify CreateRepositoryModal', () => {
  beforeAll(() => {
    repositoryFactory.registerStep(new DockerRepositoryType())
    repositoryFactory.registerStep(new HelmRepositoryType())
  })

  beforeEach(() => {
    jest.clearAllMocks()
  })

  test('should able to open create repository form and able to submit without any error', async () => {
    const { container } = render(
      <ArTestWrapper>
        <CreateRepository />
      </ArTestWrapper>
    )

    const modalDialog = await openModalDialog(container)

    const modalHeader = modalDialog.querySelector('[data-testid="modaldialog-header"]')
    expect(modalHeader).toBeInTheDocument()
    expect(modalHeader).toHaveTextContent('repositoryDetails.repositoryForm.modalTitle')
    expect(modalHeader).toHaveTextContent('repositoryDetails.repositoryForm.modalSubTitle')

    const nameField = queryByNameAttribute('identifier', modalDialog)
    expect(nameField).toBeInTheDocument()
    fireEvent.change(nameField!, { target: { value: 'repo1' } })

    const submitBtn = modalDialog.querySelector('button[aria-label="repositoryDetails.repositoryForm.create"]')
    expect(submitBtn).toBeInTheDocument()
    await userEvent.click(submitBtn!)
    await waitFor(() => {
      expect(mutateCreateRegistry).toHaveBeenCalledWith({
        body: {
          cleanupPolicy: [],
          config: { type: 'VIRTUAL', upstreamProxies: [] },
          identifier: 'repo1',
          packageType: 'DOCKER',
          parentRef: 'undefined',
          scanners: []
        },
        queryParams: { space_ref: 'undefined' }
      })
    })
  })

  test('should able to close the modal on click on Close button', async () => {
    const { container } = render(
      <ArTestWrapper>
        <CreateRepository />
      </ArTestWrapper>
    )
    const modalDialog = await openModalDialog(container)

    const cancelBtn = modalDialog.querySelector('button[aria-label="cancel"]')
    expect(cancelBtn).toBeInTheDocument()
    await userEvent.click(cancelBtn!)
    await waitFor(() => {
      expect(modalDialog).not.toBeInTheDocument()
    })

    const modalDialogAgain = await openModalDialog(container)
    const closeIcon = modalDialogAgain.querySelector('span[data-icon="Stroke"]')
    expect(closeIcon).toBeInTheDocument()
    await userEvent.click(closeIcon!)
    await waitFor(() => {
      expect(modalDialog).not.toBeInTheDocument()
    })
  })

  test('should render modal with allowed types correctly', async () => {
    const { container } = render(
      <ArTestWrapper>
        <CustomCreateRepositoryModal onSuccess={jest.fn()} allowedPackageTypes={[RepositoryPackageType.HELM]} />
      </ArTestWrapper>
    )
    const dialog = await openModalDialog(container)
    const dockerType = dialog.querySelector('input[type="checkbox"][name="packageType"][value="DOCKER"]')
    expect(dockerType).toBeDisabled()
    const helmType = dialog.querySelector('input[type="checkbox"][name="packageType"][value="HELM"]')
    expect(helmType).not.toBeDisabled()
  })
})
