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
import { fireEvent, render, waitFor } from '@testing-library/react'
import { useDeleteRegistryMutation } from '@harnessio/react-har-service-client'

import { PageType } from '@ar/common/types'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import type { VirtualRegistry } from '@ar/pages/repository-details/types'
import { MockGetDockerRegistryResponseWithAllData } from '@ar/pages/repository-details/DockerRepository/__tests__/__mockData__'

import { queryByNameAttribute } from 'utils/test/testUtils'
import DeleteRepositoryMenuItem from '../DeleteRepository'

const mutateDeleteRegistrySuccess = jest.fn().mockImplementation(
  () =>
    new Promise(onSuccess => {
      onSuccess({
        content: {
          status: 'SUCCESS'
        }
      })
    })
)

jest.mock('@harnessio/react-har-service-client', () => ({
  useDeleteRegistryMutation: jest.fn().mockImplementation(() => ({
    isLoading: false,
    mutateAsync: mutateDeleteRegistrySuccess
  }))
}))

const openModalDialog = async (): Promise<HTMLElement> => {
  const modalDialog = document.querySelector('div.bp3-dialog') as HTMLElement
  await waitFor(() => {
    expect(modalDialog).toBeInTheDocument()
  })
  return modalDialog
}

describe('Verify DeleteRepositoryModal', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  test('Verify successful delete flow when click on cancel', async () => {
    const { container } = render(
      <ArTestWrapper>
        <DeleteRepositoryMenuItem
          data={MockGetDockerRegistryResponseWithAllData.content.data as VirtualRegistry}
          readonly={false}
          pageType={PageType.Details}
        />
      </ArTestWrapper>
    )
    const actionItem = container.querySelector('span[data-icon=code-delete]')
    await userEvent.click(actionItem!)
    const dialog = await openModalDialog()
    expect(dialog).toBeInTheDocument()
    expect(dialog.querySelector('span[data-icon=danger-icon]')).toBeInTheDocument()
    expect(dialog).toHaveTextContent('repositoryList.deleteModal.title')
    expect(dialog).toHaveTextContent('repositoryList.deleteModal.contentText')

    const deleteBtn = dialog.querySelector('button[aria-label=delete]')
    expect(deleteBtn).toBeInTheDocument()
    const cancelBtn = dialog.querySelector('button[aria-label=cancel]')
    expect(cancelBtn).toBeInTheDocument()

    await userEvent.click(cancelBtn!)
    await waitFor(() => {
      expect(dialog).not.toBeInTheDocument()
    })
  })

  test('Verify successful delete flow when click on submit', async () => {
    const { container } = render(
      <ArTestWrapper>
        <DeleteRepositoryMenuItem
          data={MockGetDockerRegistryResponseWithAllData.content.data as VirtualRegistry}
          readonly={false}
          pageType={PageType.Details}
          onClose={jest.fn()}
        />
      </ArTestWrapper>
    )
    const actionItem = container.querySelector('span[data-icon=code-delete]')
    await userEvent.click(actionItem!)
    const dialog = await openModalDialog()

    const deleteBtn = dialog.querySelector('button[aria-label=delete]')
    expect(deleteBtn).toBeInTheDocument()
    const cancelBtn = dialog.querySelector('button[aria-label=cancel]')
    expect(cancelBtn).toBeInTheDocument()

    const valueField = queryByNameAttribute('value', dialog)
    fireEvent.change(valueField!, {
      target: { value: MockGetDockerRegistryResponseWithAllData.content.data.identifier }
    })

    await userEvent.click(deleteBtn!)
    await waitFor(() => {
      expect(mutateDeleteRegistrySuccess).toHaveBeenCalledWith({ registry_ref: 'undefined/docker-repo/+' })
    })
  })

  test('Verify failure delete flow when click on submit', async () => {
    const mutateDeleteRegistryFailure = jest.fn().mockImplementation(
      () =>
        new Promise((_, onFailure) => {
          onFailure({
            message: 'Filed to load data'
          })
        })
    )
    ;(useDeleteRegistryMutation as jest.Mock).mockImplementation(() => ({
      isLoading: false,
      mutateAsync: mutateDeleteRegistryFailure
    }))
    const { container } = render(
      <ArTestWrapper>
        <ArTestWrapper>
          <DeleteRepositoryMenuItem
            data={MockGetDockerRegistryResponseWithAllData.content.data as VirtualRegistry}
            readonly={false}
            pageType={PageType.Details}
          />
        </ArTestWrapper>
      </ArTestWrapper>
    )
    const actionItem = container.querySelector('span[data-icon=code-delete]')
    await userEvent.click(actionItem!)
    const dialog = await openModalDialog()

    const deleteBtn = dialog.querySelector('button[aria-label=delete]')
    expect(deleteBtn).toBeInTheDocument()
    const cancelBtn = dialog.querySelector('button[aria-label=cancel]')
    expect(cancelBtn).toBeInTheDocument()

    const valueField = queryByNameAttribute('value', dialog)
    fireEvent.change(valueField!, {
      target: { value: MockGetDockerRegistryResponseWithAllData.content.data.identifier }
    })

    await userEvent.click(deleteBtn!)
    await waitFor(() => {
      expect(mutateDeleteRegistryFailure).toHaveBeenCalledWith({ registry_ref: 'undefined/docker-repo/+' })
    })
  })
})
