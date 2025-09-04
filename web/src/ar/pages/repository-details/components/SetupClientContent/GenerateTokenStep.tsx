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

import React, { useState } from 'react'
import { defaultTo } from 'lodash-es'
import { Button, ButtonVariation, Container, getErrorInfoFromErrorObject, Text, useToaster } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import type { ClientSetupStep } from '@harnessio/react-har-service-client'

import { useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { useParentUtils } from '@ar/hooks/useParentUtils'
import CommandBlock from '@ar/components/CommandBlock/CommandBlock'

import css from './SetupClientContent.module.scss'

interface GenerateTokenStepProps {
  stepIndex: number
  step: ClientSetupStep
}
export default function GenerateTokenStep({ stepIndex, step }: GenerateTokenStepProps) {
  const [token, setToken] = useState<string>()
  const { getString } = useStrings()
  const { generateToken } = useParentUtils()
  const { useGovernanceMetaDataModal } = useParentHooks()
  const { showError, clear } = useToaster()

  const { conditionallyOpenGovernanceErrorModal } = useGovernanceMetaDataModal({
    considerWarningAsError: false,
    errorHeaderMsg: 'platform.secrets.policyEvaluations.failedToSave',
    warningHeaderMsg: 'platform.secrets.policyEvaluations.warning'
  })

  const handleGenerateToken = async () => {
    return generateToken()
      .then(res => {
        if (typeof res === 'string') {
          // For Gitness
          setToken(res)
        } else {
          // For harness enterprise
          conditionallyOpenGovernanceErrorModal(res?.metaData?.governanceMetadata, () => {
            setToken(res?.data)
          })
        }
      })
      .catch(err => {
        clear()
        showError(getErrorInfoFromErrorObject(err) || getString('repositoryDetails.clientSetup.failedToGenerateToken'))
      })
  }
  return (
    <Container className={css.stepGridContainer}>
      <Text className={css.label} font={{ variation: FontVariation.SMALL_BOLD }}>
        {getString('repositoryDetails.clientSetup.step', { stepIndex: stepIndex + 1 })}
      </Text>
      <Text flex={{ alignItems: 'center', justifyContent: 'flex-start' }} font={{ variation: FontVariation.SMALL }}>
        {step.header}
        <Button minimal variation={ButtonVariation.LINK} onClick={handleGenerateToken} icon="gitops-gnupg-key-blue">
          {token
            ? getString('repositoryDetails.clientSetup.generateNewToken')
            : getString('repositoryDetails.clientSetup.generateToken')}
        </Button>
      </Text>
      {token && (
        <>
          <div />
          <CommandBlock noWrap commandSnippet={defaultTo(token, '')} allowCopy={true} />
        </>
      )}
    </Container>
  )
}
