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

import React, { useMemo, useState } from 'react'
import {
  Button,
  ButtonVariation,
  Container,
  Dialog,
  FlexExpander,
  FormikForm,
  FormInput,
  Layout,
  SelectOption,
  Text,
  useToaster
} from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import { Formik } from 'formik'
import { useMutate } from 'restful-react'
import moment from 'moment'
import * as Yup from 'yup'
import { Else, Match, Render, Truthy } from 'react-jsx-match'
import { omit } from 'lodash-es'
import { useModalHook } from 'hooks/useModalHook'
import { useStrings } from 'framework/strings'
import type { UserCreateTokenInput } from 'services/code'
import { REGEX_VALID_REPO_NAME, getErrorMessage } from 'utils/Utils'
import { CodeIcon } from 'utils/GitUtils'
import { CopyButton } from 'components/CopyButton/CopyButton'
import { FormInputWithCopyButton } from 'components/UserManagementFlows/AddUserModal'

import css from 'components/CloneCredentialDialog/CloneCredentialDialog.module.scss'

const useNewToken = ({ onClose }: { onClose: () => void }) => {
  const { getString } = useStrings()
  const { mutate } = useMutate({ path: '/api/v1/user/tokens', verb: 'POST' })
  const { showError } = useToaster()

  const [generatedToken, setGeneratedToken] = useState<string>()
  const isTokenGenerated = Boolean(generatedToken)

  const lifeTimeOptions: SelectOption[] = useMemo(
    () => [
      { label: getString('nDays', { number: 7 }), value: 604800000000000 },
      { label: getString('nDays', { number: 30 }), value: 2592000000000000 },
      { label: getString('nDays', { number: 60 }), value: 5184000000000000 },
      { label: getString('nDays', { number: 90 }), value: 7776000000000000 },
      { label: getString('noExpiration'), value: Infinity }
    ],
    [getString]
  )

  const onModalClose = () => {
    setGeneratedToken('')
    hideModal()
    onClose()
  }

  const [openModal, hideModal] = useModalHook(() => {
    return (
      <Dialog isOpen enforceFocus={false} onClose={onModalClose} title={getString('createNewToken')}>
        <Formik<UserCreateTokenInput>
          initialValues={{
            identifier: ''
          }}
          validationSchema={Yup.object().shape({
            identifier: Yup.string()
              .required(getString('validation.nameIsRequired'))
              .matches(REGEX_VALID_REPO_NAME, getString('validation.nameInvalid')),
            lifetime: Yup.number().required(getString('validation.expirationDateRequired'))
          })}
          onSubmit={async values => {
            let payload = { ...values }

            if (payload.lifetime === Infinity) {
              payload = omit(payload, 'lifetime')
            }

            const res = await mutate(payload).catch(err => {
              showError(getErrorMessage(err))
            })
            setGeneratedToken(res?.access_token)
          }}>
          {formikProps => {
            const lifetime = formikProps.values.lifetime || 0
            const expiresAtString = moment(Date.now() + lifetime / 1000000).format('dddd, MMMM DD YYYY')

            return (
              <FormikForm>
                <FormInputWithCopyButton
                  name="identifier"
                  label={getString('name')}
                  placeholder={getString('newToken.namePlaceholder')}
                  disabled={isTokenGenerated}
                />
                <FormInput.Select
                  name="lifetime"
                  label={getString('expiration')}
                  items={lifeTimeOptions}
                  usePortal
                  disabled={isTokenGenerated}
                />
                {lifetime ? (
                  <Text
                    font={{ variation: FontVariation.SMALL_SEMI }}
                    color={Color.GREY_400}
                    margin={{ bottom: 'medium' }}>
                    {lifetime === Infinity
                      ? getString('noExpirationDate')
                      : getString('newToken.expireOn', { date: expiresAtString })}
                  </Text>
                ) : null}
                <Render when={isTokenGenerated}>
                  <Text padding={{ bottom: 'small' }} font={{ variation: FontVariation.FORM_LABEL, size: 'small' }}>
                    {getString('token')}
                  </Text>
                  <Container padding={{ bottom: 'medium' }}>
                    <Layout.Horizontal className={css.layout}>
                      <Text className={css.url}>{generatedToken}</Text>
                      <FlexExpander />
                      <CopyButton
                        content={generatedToken || ''}
                        id={css.cloneCopyButton}
                        icon={CodeIcon.Copy}
                        iconProps={{ size: 14 }}
                      />
                    </Layout.Horizontal>
                  </Container>
                  <Text padding={{ bottom: 'medium' }} font={{ variation: FontVariation.BODY2_SEMI, size: 'small' }}>
                    {getString('newToken.tokenHelptext')}
                  </Text>
                </Render>
                <Match expr={isTokenGenerated}>
                  <Truthy>
                    <Button text={getString('close')} variation={ButtonVariation.TERTIARY} onClick={onModalClose} />
                  </Truthy>
                  <Else>
                    <Layout.Horizontal margin={{ top: 'xxxlarge' }} spacing="medium">
                      <Button
                        text={getString('newToken.generateToken')}
                        type="submit"
                        variation={ButtonVariation.PRIMARY}
                      />
                      <Button text={getString('cancel')} onClick={hideModal} variation={ButtonVariation.TERTIARY} />
                    </Layout.Horizontal>
                  </Else>
                </Match>
              </FormikForm>
            )
          }}
        </Formik>
      </Dialog>
    )
  }, [generatedToken])

  return {
    openModal,
    hideModal
  }
}

export default useNewToken
