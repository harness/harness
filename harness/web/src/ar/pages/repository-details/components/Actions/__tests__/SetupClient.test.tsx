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
import { useGetClientSetupDetailsQuery } from '@harnessio/react-har-service-client'

import { PageType } from '@ar/common/types'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import type { VirtualRegistry } from '@ar/pages/repository-details/types'
import {
  MockGetDockerRegistryResponseWithAllData,
  MockGetSetupClientOnRegistryConfigPageResponse
} from '@ar/pages/repository-details/DockerRepository/__tests__/__mockData__'

import SetupClientMenuItem from '../SetupClient'

import '../../../RepositoryFactory'

jest.mock('@harnessio/react-har-service-client', () => ({
  useGetClientSetupDetailsQuery: jest.fn().mockImplementation(() => ({
    isFetching: false,
    refetch: jest.fn(),
    error: false,
    data: MockGetSetupClientOnRegistryConfigPageResponse
  }))
}))

const openModalDialog = async (className = 'div.bp3-drawer'): Promise<HTMLElement> => {
  const modalDialog = document.querySelector(className) as HTMLElement
  await waitFor(() => {
    expect(modalDialog).toBeInTheDocument()
  })
  return modalDialog
}

describe('Verify SetupClient Action', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  test('Verify successful flow', async () => {
    const { container } = render(
      <ArTestWrapper>
        <SetupClientMenuItem
          data={MockGetDockerRegistryResponseWithAllData.content.data as VirtualRegistry}
          readonly={false}
          pageType={PageType.Table}
        />
      </ArTestWrapper>
    )

    const actionItem = container.querySelector('span[data-icon=setup-client]')
    await userEvent.click(actionItem!)
    const dialog = await openModalDialog()

    const header = getByTestId(dialog, 'setup-client-header')
    expect(header).toBeInTheDocument()
    expect(header).toHaveTextContent(MockGetSetupClientOnRegistryConfigPageResponse.content.data.mainHeader)
    expect(header.querySelector('span[data-icon=docker-step]')).toBeInTheDocument()

    const footer = getByTestId(dialog, 'setup-client-footer')
    const doneBtn = footer.querySelector('button[aria-label="repositoryDetails.clientSetup.done"]')
    expect(doneBtn).toBeInTheDocument()

    await userEvent.click(doneBtn!)
    await waitFor(() => {
      expect(dialog).not.toBeInTheDocument()
    })
  })

  test('Verify failure flow', async () => {
    const refetchFn = jest.fn()
    ;(useGetClientSetupDetailsQuery as jest.Mock).mockImplementation(() => ({
      isFetching: false,
      error: { message: 'failed to load setup client steps' },
      data: null,
      refetch: refetchFn
    }))

    const { container } = render(
      <ArTestWrapper queryParams={{ tab: 'configuration' }}>
        <SetupClientMenuItem
          data={MockGetDockerRegistryResponseWithAllData.content.data as VirtualRegistry}
          readonly={false}
          pageType={PageType.Table}
        />
      </ArTestWrapper>
    )

    const actionItem = container.querySelector('span[data-icon=setup-client]')
    await userEvent.click(actionItem!)
    const dialog = await openModalDialog()

    expect(getByText(dialog, 'failed to load setup client steps')).toBeInTheDocument()
    const retryBtn = dialog.querySelector('button[aria-label=Retry]')
    expect(retryBtn).toBeInTheDocument()
    await userEvent.click(retryBtn!)
    await waitFor(() => {
      expect(refetchFn).toHaveBeenCalled()
    })
  })
})
