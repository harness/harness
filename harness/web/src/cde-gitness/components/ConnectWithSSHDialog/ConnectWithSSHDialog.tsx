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
import { Button, ButtonVariation, Card, CodeBlock, Container, Layout, ModalDialog, Text } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { useStrings } from 'framework/strings'
import css from './ConnectWithSSHDialog.module.scss'

interface ConnectWithSSHDialogProps {
  isOpen: boolean
  onClose: () => void
  connectionCommand: string
}

export const ConnectWithSSHDialog: React.FC<ConnectWithSSHDialogProps> = ({ isOpen, onClose, connectionCommand }) => {
  const { getString } = useStrings()

  return (
    <ModalDialog
      isOpen={isOpen}
      onClose={onClose}
      title={
        <Layout.Horizontal
          spacing="small"
          flex={{ alignItems: 'center', justifyContent: 'flex-start' }}
          className={css.marginBottomTitle}>
          <Text font={{ variation: FontVariation.H3 }}>{getString('cde.sshDetails.connectWithSSH')}</Text>
        </Layout.Horizontal>
      }
      className={css.modalDialog}>
      <Container
        className={css.dialogContainer}
        onClick={e => {
          e.stopPropagation()
        }}>
        <Layout.Vertical spacing="large" className={css.innerContainer}>
          <Text className={css.congratsText} color={Color.GREY_700}>
            {getString('cde.sshDetails.congratsText')}
          </Text>
          <Card className={css.contentCard}>
            <Layout.Vertical spacing="xlarge">
              <Layout.Horizontal spacing="medium" className={css.step1}>
                <Container className={css.stepNumber}>1</Container>
                <Text className={css.stepContent}> {getString('cde.sshDetails.step1')}</Text>
              </Layout.Horizontal>

              <Layout.Horizontal spacing="medium" className={css.step2}>
                <Container className={css.stepNumber}>2</Container>
                <Container>
                  <Layout.Vertical spacing="small" className={css.innerLayout}>
                    <Text className={css.stepContent}>{getString('cde.sshDetails.step2')}</Text>
                    <Container className={css.codeBlockContainer}>
                      <CodeBlock allowCopy={true} format="pre" snippet={connectionCommand} />
                    </Container>
                  </Layout.Vertical>
                </Container>
              </Layout.Horizontal>

              <Layout.Horizontal spacing="medium" className={css.step1}>
                <Container className={css.stepNumber}>3</Container>
                <Text color={Color.GREY_700} className={css.stepContent}>
                  {getString('cde.sshDetails.step3')}
                </Text>
              </Layout.Horizontal>
            </Layout.Vertical>
          </Card>

          <Layout.Horizontal className={css.buttonContainer}>
            <Button text="Done" variation={ButtonVariation.PRIMARY} onClick={onClose} />
          </Layout.Horizontal>
        </Layout.Vertical>
      </Container>
    </ModalDialog>
  )
}

export default ConnectWithSSHDialog
