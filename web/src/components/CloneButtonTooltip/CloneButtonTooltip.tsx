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

import React, { useState } from 'react'
import { Render } from 'react-jsx-match'
import cx from 'classnames'
import { Button, ButtonVariation, Container, FlexExpander, Layout, PillToggle, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { Classes } from '@blueprintjs/core'
import { Icon } from '@harnessio/icons'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { CopyButton } from 'components/CopyButton/CopyButton'
import { CodeIcon } from 'utils/GitUtils'
import CloneCredentialDialog from 'components/CloneCredentialDialog/CloneCredentialDialog'
import css from './CloneButtonTooltip.module.scss'

interface CloneButtonTooltipProps {
  httpsURL: string
  sshURL: string
}
enum CloneType {
  HTTPS = 'https',
  SSH = 'ssh'
}

export function CloneButtonTooltip({ httpsURL, sshURL }: CloneButtonTooltipProps) {
  const { getString } = useStrings()
  const [flag, setFlag] = useState(false)
  const { isCurrentSessionPublic } = useAppContext()
  const [type, setType] = useState(CloneType.HTTPS)
  return (
    <Container className={css.container} padding="xlarge">
      <Layout.Vertical spacing="small">
        <Container>
          <FlexExpander />

          <Icon
            size={16}
            name="code-close"
            className={cx(Classes.POPOVER_DISMISS, css.closeIcon)}
            onClick={() => {
              setFlag(false)
            }}
          />
        </Container>
        <Text font={{ variation: FontVariation.H4 }}>{getString('cloneHTTPS')}</Text>
        {sshURL ? (
          <Container padding={{ top: 'small' }}>
            <Layout.Vertical>
              <Container padding={{ bottom: 'medium' }}>
                <PillToggle
                  selectedView={type}
                  onChange={typeClicked => {
                    setType(typeClicked as CloneType)
                  }}
                  options={[
                    { label: CloneType.HTTPS.toUpperCase(), value: 'https' },
                    { label: CloneType.SSH.toUpperCase(), value: 'ssh' }
                  ]}
                />
              </Container>
              {type === CloneType.HTTPS ? (
                <Layout.Horizontal className={css.layout}>
                  <Text className={css.url}>{httpsURL}</Text>

                  <CopyButton
                    content={httpsURL}
                    id={css.cloneCopyButton}
                    icon={CodeIcon.Copy}
                    iconProps={{ size: 14 }}
                  />
                </Layout.Horizontal>
              ) : (
                <Layout.Horizontal className={css.layout}>
                  <Text className={css.url}>{sshURL}</Text>

                  <CopyButton content={sshURL} id={css.cloneCopyButton} icon={CodeIcon.Copy} iconProps={{ size: 14 }} />
                </Layout.Horizontal>
              )}
            </Layout.Vertical>
          </Container>
        ) : (
          <Container padding={{ top: 'small' }}>
            <Layout.Horizontal className={css.layout}>
              <Text className={css.url}>{httpsURL}</Text>

              <CopyButton content={httpsURL} id={css.cloneCopyButton} icon={CodeIcon.Copy} iconProps={{ size: 14 }} />
            </Layout.Horizontal>
          </Container>
        )}

        {type === CloneType.HTTPS ? (
          <Render when={!isCurrentSessionPublic}>
            <Button
              width={300}
              onClick={() => {
                setFlag(true)
              }}
              variation={ButtonVariation.SECONDARY}>
              {getString('generateCloneCred')}
            </Button>
            <Text
              padding={{ top: 'small' }}
              width={300}
              icon={'code-info'}
              className={css.codeText}
              iconProps={{ size: 16 }}
              color={Color.GREY_700}
              font={{ variation: FontVariation.BODY2_SEMI, size: 'xsmall' }}>
              {getString('generateCloneText')}
            </Text>
          </Render>
        ) : null}
      </Layout.Vertical>
      <CloneCredentialDialog flag={flag} setFlag={setFlag} />
    </Container>
  )
}
