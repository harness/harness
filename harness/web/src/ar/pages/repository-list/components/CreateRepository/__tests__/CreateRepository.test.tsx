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
import userEvent from '@testing-library/user-event'

import { testSelectChange } from '@ar/utils/testUtils/utils'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'

import { CreateRepository } from '../CreateRepository'

const mockHistoryPush = jest.fn()
// eslint-disable-next-line jest-no-mock
jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useHistory: () => ({
    push: mockHistoryPush
  })
}))

jest.mock('@ar/pages/repository-details/hooks/useCreateRepositoryModal/useCreateRepositoryModal', () => ({
  useCreateRepositoryModal: jest.fn().mockImplementation(({ onSuccess }) => [() => onSuccess({ identifier: 'repo1' })])
}))

jest.mock('@ar/pages/upstream-proxy-details/hooks/useCreateUpstreamProxyModal/useCreateUpstreamProxyModal', () => ({
  __esModule: true,
  default: jest.fn().mockImplementation(({ onSuccess }) => [() => onSuccess({ identifier: 'upstreamRepo1' })])
}))

describe('Verify CreateRepository component', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  test('Verify callbacks are being called correctly', async () => {
    const { container } = render(
      <ArTestWrapper>
        <CreateRepository />
      </ArTestWrapper>
    )

    const mainBtn = container.querySelector('button[aria-label="repositoryList.newRepository"]')
    expect(mainBtn!).toBeInTheDocument()
    await userEvent.click(mainBtn!)
    expect(mockHistoryPush).toHaveBeenLastCalledWith('/registries/repo1/configuration')

    const dropDownBtn = container.querySelector('span[icon="chevron-down"]') as HTMLElement
    await testSelectChange(dropDownBtn!, 'repositoryList.upstreamProxy.label', '', 'bp3-popover')
    expect(mockHistoryPush).toHaveBeenLastCalledWith('/registries/upstreamRepo1/configuration')
  })
})
