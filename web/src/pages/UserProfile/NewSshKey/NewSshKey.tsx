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
import { Button, ButtonVariation, Dialog, FormikForm, FormInput, Layout, Text, useToaster } from '@harnessio/uicore'
import { Formik } from 'formik'
import { useMutate } from 'restful-react'
import * as Yup from 'yup'

import { PopoverInteractionKind, PopoverPosition } from '@blueprintjs/core'
import { useModalHook } from 'hooks/useModalHook'
import { useStrings } from 'framework/strings'
import { EnumPublicKeyUsage, REGEX_VALID_REPO_NAME, getErrorMessage } from 'utils/Utils'

import { FormInputWithCopyButton } from 'components/UserManagementFlows/AddUserModal'

import css from '../UserProfile.module.scss'

interface userSshKeyRequest {
  content: string
  identifier: string
  usage: EnumPublicKeyUsage
}
const useNewSshKey = ({ onClose }: { onClose: () => void }) => {
  const { getString } = useStrings()
  const { mutate } = useMutate({ path: '/api/v1/user/keys', verb: 'POST' })
  const { showError } = useToaster()

  const [generatedToken, setGeneratedToken] = useState<string>()
  const isTokenGenerated = Boolean(generatedToken)

  //   const lifeTimeOptions: SelectOption[] = useMemo(
  //     () => [
  //       { label: getString('nDays', { number: 7 }), value: 604800000000000 },
  //       { label: getString('nDays', { number: 30 }), value: 2592000000000000 },
  //       { label: getString('nDays', { number: 60 }), value: 5184000000000000 },
  //       { label: getString('nDays', { number: 90 }), value: 7776000000000000 },
  //       { label: getString('noExpiration'), value: Infinity }
  //     ],
  //     [getString]
  //   )

  const onModalClose = () => {
    setGeneratedToken('')
    hideSshKeyModal()
    onClose()
  }

  const [openSshKeyModal, hideSshKeyModal] = useModalHook(() => {
    return (
      <Dialog isOpen enforceFocus={false} onClose={onModalClose} title={getString('sshCard.newSshKey')}>
        <Formik<userSshKeyRequest>
          initialValues={{
            identifier: '',
            content: '',
            usage: 'auth'
          }}
          validationSchema={Yup.object().shape({
            identifier: Yup.string()
              .required(getString('validation.nameIsRequired'))
              .matches(REGEX_VALID_REPO_NAME, getString('validation.nameInvalid'))
          })}
          onSubmit={async values => {
            const payload = { ...values }
            mutate(payload)
              .then(() => {
                hideSshKeyModal()
                onClose()
              })
              .catch(err => {
                showError(getErrorMessage(err))
              })
          }}>
          {() => {
            return (
              <FormikForm>
                <FormInputWithCopyButton
                  name="identifier"
                  label={getString('sshCard.newSshKey')}
                  placeholder={getString('newToken.namePlaceholder')}
                  disabled={isTokenGenerated}
                />
                <FormInput.TextArea
                  label={
                    <Layout.Horizontal
                      className={css.textAreaContainer}
                      flex={{ justifyContent: 'flex-start', alignItems: 'flex-start' }}>
                      <Text>{getString('sshCard.publicKey')}</Text>

                      <Text
                        padding={{ left: 'small' }}
                        className={css.icon}
                        icon="code-info"
                        tooltip={getString('sshCard.beginsWithContent')}
                        tooltipProps={{
                          isDark: true,
                          interactionKind: PopoverInteractionKind.HOVER,
                          position: PopoverPosition.BOTTOM
                        }}
                        iconProps={{ size: 16 }}
                      />
                    </Layout.Horizontal>
                  }
                  name="content"
                />

                <Layout.Horizontal margin={{ top: 'xxxlarge' }} spacing="medium">
                  <Button text={getString('save')} type="submit" variation={ButtonVariation.PRIMARY} />
                  <Button text={getString('cancel')} onClick={hideSshKeyModal} variation={ButtonVariation.TERTIARY} />
                </Layout.Horizontal>
              </FormikForm>
            )
          }}
        </Formik>
      </Dialog>
    )
  }, [generatedToken])

  return {
    openSshKeyModal,
    hideSshKeyModal
  }
}

export default useNewSshKey
