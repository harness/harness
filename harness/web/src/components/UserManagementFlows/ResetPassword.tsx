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
import {
  Avatar,
  Button,
  ButtonVariation,
  Dialog,
  FlexExpander,
  Layout,
  StringSubstitute,
  Text,
  useToaster
} from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { Else, Match, Truthy } from 'react-jsx-match'
import { useModalHook } from 'hooks/useModalHook'
import { useStrings } from 'framework/strings'
import { TypesUser, useAdminUpdateUser } from 'services/code'
import { generateAlphaNumericHash, getErrorMessage } from 'utils/Utils'

import { GeneratedPassword } from './AddUserModal'

import css from './UserManagementFlows.module.scss'

const useResetPasswordModal = () => {
  const { getString } = useStrings()
  const { showError } = useToaster()

  const [userDetails, setUserDetails] = useState<TypesUser>()
  const [newPassword, setNewPassword] = useState('')

  const { mutate: updateUser } = useAdminUpdateUser({
    user_uid: userDetails?.uid || ''
  })

  const onConfirm = async () => {
    try {
      const password = generateAlphaNumericHash(10)

      await updateUser({ password })
      setNewPassword(password)
    } catch (error) {
      showError(getErrorMessage(error))
    }
  }

  const onClose = () => {
    hideModal()
    setNewPassword('')
  }

  const [openModal, hideModal] = useModalHook(
    () => (
      <Dialog isOpen enforceFocus={false} onClose={onClose} title={'Reset Password'} chidrenClassName={css.dialogCtn}>
        <Layout.Vertical height="100%">
          <Match expr={newPassword}>
            <Truthy>
              <GeneratedPassword password={newPassword} />
              <FlexExpander />
              <div>
                <Button
                  margin={{ top: 'xxxlarge' }}
                  text={getString('close')}
                  variation={ButtonVariation.TERTIARY}
                  onClick={onClose}
                />
              </div>
            </Truthy>
            <Else>
              <Text font={{ variation: FontVariation.BODY2 }}>
                <StringSubstitute
                  str={getString('userManagement.resetPasswordMsg', {
                    displayName: userDetails?.display_name,
                    userId: userDetails?.uid
                  })}
                  vars={{
                    avatar: <Avatar name={userDetails?.display_name} />
                  }}
                />
              </Text>
              <FlexExpander />
              <div>
                <Button
                  margin={{ top: 'xxxlarge' }}
                  onClick={onConfirm}
                  text={'Confirm'}
                  variation={ButtonVariation.PRIMARY}
                />
              </div>
            </Else>
          </Match>
        </Layout.Vertical>
      </Dialog>
    ),
    [onConfirm, userDetails, newPassword]
  )

  return {
    openModal: ({ userInfo }: { userInfo?: TypesUser }) => {
      openModal()
      setUserDetails(userInfo)
    }
  }
}

export default useResetPasswordModal
