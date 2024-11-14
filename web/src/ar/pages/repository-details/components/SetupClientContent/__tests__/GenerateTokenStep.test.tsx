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
import { render, waitFor } from '@testing-library/react'
import type { ClientSetupStep } from '@harnessio/react-har-service-client'

import { useParentUtils } from '@ar/hooks/useParentUtils'
import ArTestWrapper from '@ar/utils/testUtils/ArTestWrapper'
import { MockGetSetupClientOnRegistryConfigPageResponse } from '@ar/pages/repository-details/DockerRepository/__tests__/__mockData__'

import GenerateTokenStep from '../GenerateTokenStep'

const generateTokenSuccess = jest.fn().mockImplementation(() => new Promise(onSuccess => onSuccess('12,3,452')))

jest.mock('@ar/hooks/useParentUtils', () => ({
  useParentUtils: jest.fn().mockImplementation(() => ({
    generateToken: generateTokenSuccess
  }))
}))

describe('Verify GenerateTokenStep', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  test('Verify generate token action with successful response', async () => {
    const step = MockGetSetupClientOnRegistryConfigPageResponse.content.data.sections[0].steps[1]
    const { container } = render(
      <ArTestWrapper>
        <GenerateTokenStep stepIndex={0} step={step as ClientSetupStep} />
      </ArTestWrapper>
    )

    expect(container).toHaveTextContent(step.header)
    const btn = container.querySelector('button')
    expect(btn).toHaveTextContent('repositoryDetails.clientSetup.generateToken')
    await userEvent.click(btn!)
    await waitFor(() => {
      expect(generateTokenSuccess).toHaveBeenCalledWith()
    })
    expect(btn).toHaveTextContent('repositoryDetails.clientSetup.generateNewToken')
  })

  test('Verify generate token action with failure response', async () => {
    const generateTokenFailure = jest
      .fn()
      .mockImplementation(() => new Promise((_, onFailure) => onFailure({ message: 'failed to generate token' })))

    ;(useParentUtils as jest.Mock).mockImplementation(() => ({
      generateToken: generateTokenFailure
    }))

    const step = MockGetSetupClientOnRegistryConfigPageResponse.content.data.sections[0].steps[1]
    const { container } = render(
      <ArTestWrapper>
        <GenerateTokenStep stepIndex={0} step={step as ClientSetupStep} />
      </ArTestWrapper>
    )

    expect(container).toHaveTextContent(step.header)
    const btn = container.querySelector('button')
    expect(btn).toHaveTextContent('repositoryDetails.clientSetup.generateToken')
    await userEvent.click(btn!)
    await waitFor(() => {
      expect(generateTokenFailure).toHaveBeenCalledWith()
    })
  })
})
