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
import { getByTestId, getByText, render, waitFor } from '@testing-library/react'

import { PageType } from '@ar/common/types'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import type { VirtualRegistry } from '@ar/pages/repository-details/types'
import {
  MockGetDockerRegistryResponseWithAllData,
  MockGetSetupClientOnRegistryConfigPageResponse
} from '@ar/pages/repository-details/DockerRepository/__tests__/__mockData__'

import RepositoryActions from '../RepositoryActions'

import '../../../RepositoryFactory'

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
  })),
  useGetClientSetupDetailsQuery: jest.fn().mockImplementation(() => ({
    isFetching: false,
    refetch: jest.fn(),
    error: false,
    data: MockGetSetupClientOnRegistryConfigPageResponse
  }))
}))

const openModalDialog = async (className = 'div.bp3-dialog'): Promise<HTMLElement> => {
  const modalDialog = document.querySelector(className) as HTMLElement
  await waitFor(() => {
    expect(modalDialog).toBeInTheDocument()
  })
  return modalDialog
}

describe('Verify RepositoryActions', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  test('Verify Delete action', async () => {
    const { container } = render(
      <ArTestWrapper>
        <RepositoryActions
          data={MockGetDockerRegistryResponseWithAllData.content.data as VirtualRegistry}
          readonly={false}
          pageType={PageType.Table}
        />
      </ArTestWrapper>
    )

    const actionBtn = container.querySelector('span[data-icon=Options]')
    expect(actionBtn).toBeInTheDocument()

    await userEvent.click(actionBtn!)
    const actionsList = await openModalDialog('div.bp3-popover')
    const deleteIcon = actionsList.querySelector('span[data-icon=code-delete]')
    expect(deleteIcon).toBeInTheDocument()
    await userEvent.click(deleteIcon!)

    const deleteModal = await openModalDialog()
    expect(getByText(deleteModal, 'repositoryList.deleteModal.title')).toBeInTheDocument()
    const deleteBtn = deleteModal.querySelector('button[aria-label=delete]')
    expect(deleteBtn).toBeInTheDocument()
    const cancelBtn = deleteModal.querySelector('button[aria-label=cancel]')
    expect(cancelBtn).toBeInTheDocument()

    await userEvent.click(deleteBtn!)
    await waitFor(() => {
      expect(mutateDeleteRegistrySuccess).toHaveBeenCalledWith({ registry_ref: 'undefined/docker-repo/+' })
      expect(deleteModal).not.toBeInTheDocument()
    })
  })

  test('Verify SetupClient action', async () => {
    const { container } = render(
      <ArTestWrapper>
        <RepositoryActions
          data={MockGetDockerRegistryResponseWithAllData.content.data as VirtualRegistry}
          readonly={false}
          pageType={PageType.Table}
        />
      </ArTestWrapper>
    )
    const actionBtn = container.querySelector('span[data-icon=Options]')
    expect(actionBtn).toBeInTheDocument()

    await userEvent.click(actionBtn!)
    const actionsList = await openModalDialog('div.bp3-popover')
    const setupClientIcon = actionsList.querySelector('span[data-icon=setup-client]')
    expect(setupClientIcon).toBeInTheDocument()
    await userEvent.click(setupClientIcon!)
    const setupClientDialog = await openModalDialog('div.bp3-drawer')

    const header = getByTestId(setupClientDialog, 'setup-client-header')
    expect(header).toBeInTheDocument()
    expect(header).toHaveTextContent(MockGetSetupClientOnRegistryConfigPageResponse.content.data.mainHeader)
    expect(header.querySelector('span[data-icon=docker-step]')).toBeInTheDocument()

    const footer = getByTestId(setupClientDialog, 'setup-client-footer')
    const doneBtn = footer.querySelector('button[aria-label="repositoryDetails.clientSetup.done"]')
    expect(doneBtn).toBeInTheDocument()

    await userEvent.click(doneBtn!)
    await waitFor(() => {
      expect(setupClientDialog).not.toBeInTheDocument()
    })
  })
})
