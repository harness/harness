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

import { RepositoryConfigType } from '@ar/common/types'
import { testSelectChange } from '@ar/utils/testUtils/utils'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'

import CreateRepositoryButton from '../CreateRepositoryButton'

describe('Verify CreateRepositoryButton flow', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  test('Verify callbacks are being called correctly', async () => {
    const callback = jest.fn()
    const { container } = render(
      <ArTestWrapper>
        <CreateRepositoryButton onClick={callback} />
      </ArTestWrapper>
    )

    const mainBtn = container.querySelector('button[aria-label="repositoryList.newRepository"]')
    expect(mainBtn!).toBeInTheDocument()
    await userEvent.click(mainBtn!)
    expect(callback).toHaveBeenLastCalledWith(RepositoryConfigType.VIRTUAL)

    const dropDownBtn = container.querySelector('span[icon="chevron-down"]') as HTMLElement
    expect(dropDownBtn).toBeInTheDocument()
    await testSelectChange(dropDownBtn!, 'repositoryList.artifactRegistry.label', '', 'bp3-popover')
    expect(callback).toHaveBeenLastCalledWith(RepositoryConfigType.VIRTUAL)

    await testSelectChange(dropDownBtn!, 'repositoryList.upstreamProxy.label', '', 'bp3-popover')
    expect(callback).toHaveBeenLastCalledWith(RepositoryConfigType.UPSTREAM)
  })
})
