/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { Button, ButtonVariation, Container, Dialog, FlexExpander, Layout, Text, useToaster } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { useMutate } from 'restful-react'
import { useHistory } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { CopyButton } from 'components/CopyButton/CopyButton'
import { CodeIcon } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import { generateAlphaNumericHash } from 'utils/Utils'
import css from './CloneCredentialDialog.module.scss'

interface CloneCredentialDialogProps {
  setFlag: (val: boolean) => void
  flag: boolean
}

const CloneCredentialDialog = (props: CloneCredentialDialogProps) => {
  const { setFlag, flag } = props
  const history = useHistory()
  const { getString } = useStrings()
  const { hooks, currentUser, currentUserProfileURL, standalone, routes } = useAppContext()
  const [token, setToken] = useState('')
  const { showError } = useToaster()
  const hash = generateAlphaNumericHash(6)
  const { mutate } = useMutate({ path: '/api/v1/user/tokens', verb: 'POST' })
  const genToken = useCallback(
    async (_props: { uid: string }) => {
      const res = await mutate({ uid: _props.uid })
      try {
        setToken(res?.access_token)
      } catch {
        showError(res?.data?.message || res?.message)
      }
      return res
    },
    [mutate, showError]
  )
  const tokenData = standalone ? false : hooks?.useGenerateToken?.(hash, currentUser?.uid, flag)
  const displayName = useMemo(() => currentUser?.display_name?.split('@')[0] || '', [currentUser])

  useEffect(() => {
    if (tokenData) {
      if (tokenData.status >= 400 && flag) {
        showError(tokenData.data.message || tokenData.message)
      } else {
        setToken(tokenData.data)
      }
    } else if (!tokenData && standalone && flag) {
      genToken({ uid: `code_token_${hash}` })
    }
  }, [flag, tokenData, showError]) // eslint-disable-line react-hooks/exhaustive-deps
  return (
    <Dialog
      isOpen={flag}
      enforceFocus={false}
      onClose={() => {
        setFlag(false)
      }}
      title={
        <Text font={{ variation: FontVariation.H3 }} icon={'success-tick'} iconProps={{ size: 26 }}>
          {getString('getMyCloneTitle')}
        </Text>
      }
      style={{ width: 490, maxHeight: '95vh', overflow: 'auto' }}>
      <Layout.Vertical width={380}>
        <Text padding={{ bottom: 'small' }} font={{ variation: FontVariation.FORM_LABEL, size: 'small' }}>
          {getString('userName')}
        </Text>
        <Container padding={{ bottom: 'medium' }}>
          <Layout.Horizontal className={css.layout}>
            <Text className={css.url}>{displayName}</Text>
            <FlexExpander />
            <CopyButton content={displayName} id={css.cloneCopyButton} icon={CodeIcon.Copy} iconProps={{ size: 14 }} />
          </Layout.Horizontal>
        </Container>
        <Text padding={{ bottom: 'small' }} font={{ variation: FontVariation.FORM_LABEL, size: 'small' }}>
          {getString('passwordApi')}
        </Text>

        <Container padding={{ bottom: 'medium' }}>
          <Layout.Horizontal className={css.layout}>
            <Text className={css.url}>{token}</Text>
            <FlexExpander />
            <CopyButton content={token} id={css.cloneCopyButton} icon={CodeIcon.Copy} iconProps={{ size: 14 }} />
          </Layout.Horizontal>
        </Container>
        <Text padding={{ bottom: 'medium' }} font={{ variation: FontVariation.BODY2_SEMI, size: 'small' }}>
          {getString('cloneText')}
        </Text>
        <Button
          onClick={() => {
            history.push(standalone ? routes.toCODEUserProfile() : currentUserProfileURL)
          }}
          variation={ButtonVariation.TERTIARY}
          text={getString('manageApiToken')}
        />
      </Layout.Vertical>
    </Dialog>
  )
}

export default CloneCredentialDialog
