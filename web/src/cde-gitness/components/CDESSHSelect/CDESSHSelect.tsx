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
import * as Yup from 'yup'
import {
  Button,
  ButtonVariation,
  Container,
  Dialog,
  Formik,
  FormikForm,
  FormInput,
  Layout,
  Text,
  useToaster
} from '@harnessio/uicore'
import { useFormikContext } from 'formik'
import { Menu, MenuItem } from '@blueprintjs/core'
import { Color } from '@harnessio/design-system'
import { useAppContext } from 'AppContext'
import Secret from 'cde-gitness/assests/secret.svg?url'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import type { OpenapiCreateGitspaceRequest } from 'services/cde'
import { useModalHook } from 'hooks/useModalHook'
import { useStrings } from 'framework/strings'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { getErrorMessage } from 'utils/Utils'
import { CDECustomDropdown } from '../CDECustomDropdown/CDECustomDropdown'
import { getSelectedExpirationDate } from './CDESSHSelect.utils'
import css from './CDESSHSelect.module.scss'

export const CDESSHSelect = () => {
  const { getString } = useStrings()
  const { showError } = useToaster()
  const { values, setFieldValue } = useFormikContext<OpenapiCreateGitspaceRequest>()
  const { accountIdentifier } = useGetCDEAPIParams()
  const { hooks, currentUser } = useAppContext()
  const { useListAggregatedTokens, useDeleteToken, useCreateToken } = hooks
  const { data, loading, refetch } = useListAggregatedTokens({
    queryParams: {
      accountIdentifier,
      apiKeyType: 'SSH_KEY',
      parentIdentifier: currentUser.uid
    }
  })

  const { mutate: createToken } = useCreateToken({
    queryParams: { accountIdentifier }
  })

  const { mutate: deleteToken } = useDeleteToken({
    queryParams: {
      accountIdentifier,
      apiKeyType: 'SSH_KEY',
      parentIdentifier: currentUser.uid,
      apiKeyIdentifier: `cdesshkey`
    }
  })

  const tokenList = data?.data?.content.map((item: { token: any }) => item.token) || []

  const [openModal, hideModal] = useModalHook(() => {
    const expiryOptions = [
      { label: getString('cde.sshSelect.30days'), value: '30' },
      { label: getString('cde.sshSelect.90days'), value: '90' },
      { label: getString('cde.sshSelect.180days'), value: '180' },
      { label: getString('cde.sshSelect.noexpiration'), value: '-1' }
    ]

    return (
      <Dialog isOpen onClose={hideModal} title="Add SSH Key" className={css.main}>
        <Layout.Vertical spacing="large">
          <Text margin="larg">Add an SSH key for secure access to Gitspaces via SSH.</Text>
          <Layout.Horizontal className={css.note} spacing="small">
            <Text icon="info-messaging" font="small" iconProps={{ size: 24 }}>
              SSH key are used to connect securely to workspaces
            </Text>
            <Text font="small" color={Color.PRIMARY_5}>
              Learn how to create an SSH Key
            </Text>
          </Layout.Horizontal>
          <Formik<{
            sshKeyName: string
            sshKeyValue: string
            expiryDate: string
            expiry: string
          }>
            formName="sshCreate"
            onSubmit={async value => {
              try {
                await createToken({
                  identifier: value?.sshKeyName.trim(),
                  name: value?.sshKeyName.trim(),
                  description: '',
                  tags: {},
                  accountIdentifier,
                  apiKeyIdentifier: `cdesshkey`,
                  parentIdentifier: currentUser.uid,
                  apiKeyType: 'SSH_KEY',
                  sshKeyContent: value.sshKeyValue,
                  sshKeyUsage: ['AUTH'],
                  validTo: Date.parse(value?.expiryDate),
                  validFrom: new Date().getTime()
                })
                setFieldValue('ssh_token_identifier', value?.sshKeyName.trim())
                hideModal()
                refetch()
              } catch (error) {
                showError(getErrorMessage(error))
                hideModal()
              }
            }}
            initialValues={{
              sshKeyName: '',
              sshKeyValue: '',
              expiryDate: getSelectedExpirationDate('30'),
              expiry: '30'
            }}
            validationSchema={Yup.object().shape({
              sshKeyName: Yup.string().required(),
              sshKeyValue: Yup.string().required()
            })}>
            {formikProps => {
              return (
                <FormikForm>
                  <FormInput.Text name="sshKeyName" label="Key Name" />
                  <FormInput.DropDown
                    items={expiryOptions}
                    name="expiry"
                    dropDownProps={{
                      filterable: false,
                      minWidth: 100
                    }}
                    label={getString('expirationDate')}
                    onChange={item => {
                      formikProps.setFieldValue('expiryDate', getSelectedExpirationDate(item.value.toString()))
                    }}
                  />
                  <FormInput.TextArea
                    placeholder={`Begins with 'ssh-rsa', 'ecdsa-sha2-nistp256', 'ecdsa-sha2-nistp384', 'ecdsa-sha2-nistp521', 'ssh-ed25519', 'sk-ecdsa-sha2-nistp256@openssh.com', or 'sk-ssh-ed25519@openssh.com'`}
                    name="sshKeyValue"
                    label="SSH Key"
                    className={css.sshKeyValue}
                  />
                  <Layout.Horizontal spacing="large">
                    <Button variation={ButtonVariation.PRIMARY} type="submit">
                      Add Key
                    </Button>
                    <Button onClick={hideModal} variation={ButtonVariation.TERTIARY}>
                      Cancel
                    </Button>
                  </Layout.Horizontal>
                </FormikForm>
              )
            }}
          </Formik>
        </Layout.Vertical>
      </Dialog>
    )
  }, [accountIdentifier])

  const confirmDelete = useConfirmAct()

  const handleDelete = async (e: React.MouseEvent<HTMLElement, MouseEvent>, tokenId: string): Promise<void> => {
    e.stopPropagation()
    confirmDelete({
      intent: 'danger',
      title: getString('cde.sshSelect.deleteToken', { name: tokenId }),
      message: getString('cde.deleteGitspaceText'),
      confirmText: getString('delete'),
      action: async () => {
        try {
          await deleteToken(tokenId || '', {
            headers: { 'content-type': 'application/json' }
          })
          refetch()
        } catch (err) {
          showError(getErrorMessage(err))
        }
      }
    })
  }

  return (
    <CDECustomDropdown
      leftElement={
        <Layout.Horizontal>
          <img src={Secret} height={20} width={20} style={{ marginRight: '8px', alignItems: 'center' }} />
          <Layout.Vertical spacing="small">
            <Text>SSH Key </Text>
            <Text font="small" width="56%">
              By default we will create the SSH key used to login to the Gitspace. You can add keys under Preferences in
              User Settings
            </Text>
          </Layout.Vertical>
        </Layout.Horizontal>
      }
      label={
        <Text icon={loading ? 'loading' : undefined}>{values?.ssh_token_identifier || '-- Select SSH Key --'}</Text>
      }
      menu={
        <Menu>
          <Container border={{ bottom: true }}>
            {tokenList.length ? (
              <>
                {tokenList.map((item: { name: string; identifier: string; apiKeyIdentifier: string }) => {
                  return (
                    <MenuItem
                      key={item.identifier}
                      text={
                        <Text
                          width="100%"
                          flex={{ justifyContent: 'space-between' }}
                          rightIcon="cross"
                          rightIconProps={{
                            size: 12,
                            onClick: e => {
                              handleDelete(e, item.identifier)
                              setFieldValue('ssh_token_identifier', undefined)
                            }
                          }}>
                          {item.name}
                        </Text>
                      }
                      onClick={() => {
                        setFieldValue('ssh_token_identifier', item.identifier)
                      }}
                    />
                  )
                })}
              </>
            ) : (
              <Text padding="small">
                There are no keys configured. By default we will create a SSH key to login into Gitspace.
              </Text>
            )}
          </Container>
          <Button
            variation={ButtonVariation.LINK}
            onClick={() => {
              openModal()
            }}
            className={css.addsshButton}>
            Add SSH Key
          </Button>
        </Menu>
      }
    />
  )
}
