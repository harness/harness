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

import React, { useMemo } from 'react'
import {
  Avatar,
  Button,
  ButtonSize,
  ButtonVariation,
  Card,
  Container,
  Layout,
  Page,
  TableV2,
  Text,
  useToaster
} from '@harnessio/uicore'
import { useAtom } from 'jotai'
import { Color, FontVariation } from '@harnessio/design-system'
import { useHistory } from 'react-router-dom'
import { useGet, useMutate } from 'restful-react'
import type { CellProps, Column } from 'react-table'
import ReactTimeago from 'react-timeago'
import moment from 'moment'
import { defaultTo, isEmpty } from 'lodash-es'
import { useStrings } from 'framework/strings'
import { TypesToken, TypesUser, useGetUser, useOpLogout, useUpdateUser } from 'services/code'
import { ButtonRoleProps, TypeKeys, getErrorMessage } from 'utils/Utils'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { useAppContext } from 'AppContext'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { currentUserAtom } from 'atoms/currentUser'
import { TimePopoverWithLocal } from 'utils/timePopoverLocal/TimePopoverWithLocal'
import useNewToken from './NewToken/NewToken'
import EditableTextField from './EditableTextField'
import SshKey from '../../icons/sshKey.svg?url'

import useNewSshKey from './NewSshKey/NewSshKey'
import css from './UserProfile.module.scss'

const USER_TOKENS_API_PATH = '/api/v1/user/tokens'

const UserProfile = () => {
  const history = useHistory()
  const { getString } = useStrings()
  const { showSuccess, showError } = useToaster()
  const { routes } = useAppContext()

  const { data: currentUser, loading: currentUserLoading, refetch: refetchCurrentUser } = useGetUser({})
  const { mutate: updateUser } = useUpdateUser({})
  const { mutate: logoutUser } = useOpLogout({})

  const { data: userTokens, loading: tokensLoading, refetch: refetchTokens } = useGet({ path: USER_TOKENS_API_PATH })
  const { data: sshKeys, loading: sshKeysLoading, refetch: refetchSshKeys } = useGet({ path: '/api/v1/user/keys' })
  const { mutate: deleteSshKey } = useMutate({ path: `/api/v1/user/keys`, verb: 'DELETE' })

  const { mutate: deleteToken } = useMutate({ path: USER_TOKENS_API_PATH, verb: 'DELETE' })
  const [, setCurrentUser] = useAtom(currentUserAtom)

  const onLogout = async () => {
    await logoutUser()
    history.push(routes.toSignIn())
    setCurrentUser(undefined)
  }

  const { data: systemConfig, loading: systemConfigLoading } = useGet({ path: 'api/v1/system/config' })

  const { openModal } = useNewToken({ onClose: refetchTokens })
  const { openSshKeyModal } = useNewSshKey({ onClose: refetchSshKeys })

  const onConfirmAct = useConfirmAct()
  const handleDeleteSshKey = async (sshKeyId: string) =>
    await onConfirmAct({
      action: async () => {
        try {
          await deleteSshKey(sshKeyId)
          refetchSshKeys()
          showSuccess(getString('sshCard.successSshKeyMsg'))
        } catch (error) {
          showError(getErrorMessage(error))
        }
      },
      message: getString('sshCard.deleteSshMsg'),
      intent: 'danger',
      title: getString('sshCard.deleteSshTitle')
    })

  const handleDeleteToken = async (tokenId: string) =>
    await onConfirmAct({
      action: async () => {
        try {
          await deleteToken(tokenId)
          refetchTokens()
          showSuccess(getString('deleteTokenMsg'))
        } catch (error) {
          showError(getErrorMessage(error))
        }
      },
      message: getString('userProfile.deleteTokenMsg'),
      intent: 'danger',
      title: getString('deleteToken')
    })

  // useEffect(()=>{},[refetchSshKeys])
  const sshkeysColumns: Column<TypeKeys>[] = useMemo(
    () => [
      {
        Header: getString('name'),
        width: '65%',
        Cell: ({ row }: CellProps<TypeKeys>) => {
          return (
            <Layout.Horizontal flex={{ justifyContent: 'flex-start' }}>
              <Container className={css.iconContainer} padding="small">
                <img src={SshKey} width={16} height={16} />
              </Container>
              <Layout.Vertical>
                <Text font={{ variation: FontVariation.CARD_TITLE }} lineClamp={1}>
                  {row.original.identifier}
                </Text>

                <Text font={{ variation: FontVariation.BODY2_SEMI }} lineClamp={1}>
                  {row.original.fingerprint}
                </Text>
              </Layout.Vertical>
            </Layout.Horizontal>
          )
        }
      },
      {
        Header: getString('sshCard.addedOn'),
        width: '15%',
        Cell: ({ row }: CellProps<TypeKeys>) => (
          <Text font={{ variation: FontVariation.SMALL_SEMI }} lineClamp={1}>
            <TimePopoverWithLocal time={defaultTo(row.original?.created, 0)} />
          </Text>
        )
      },
      {
        Header: getString('sshCard.lastUsed'),
        width: '15%',
        Cell: ({ row }: CellProps<TypeKeys>) => (
          <Text font={{ variation: FontVariation.SMALL_SEMI }} lineClamp={1}>
            <TimePopoverWithLocal time={defaultTo(row.original?.verified, 0)} />
          </Text>
        )
      },
      {
        accessor: 'identifier',
        Header: '',
        width: '5%',
        Cell: ({ row }: CellProps<TypeKeys>) => {
          return (
            <OptionsMenuButton
              tooltipProps={{ isDark: true }}
              items={[
                {
                  text: getString('sshCard.deleteSshTitle'),
                  onClick: () => {
                    handleDeleteSshKey(row.original.identifier as string)
                    refetchSshKeys()
                  }
                }
              ]}
            />
          )
        }
      }
    ],
    [sshKeys] // eslint-disable-line react-hooks/exhaustive-deps
  )
  const columns: Column<TypesToken>[] = useMemo(
    () => [
      {
        Header: getString('token'),
        width: '25%',
        Cell: ({ row }: CellProps<TypesToken>) => {
          return (
            <Text font={{ variation: FontVariation.SMALL_SEMI }} lineClamp={1}>
              {row.original.identifier}
            </Text>
          )
        }
      },
      {
        Header: getString('status'),
        width: '20%',
        Cell: ({ row }: CellProps<TypesToken>) => {
          const isActive = !row.original.expires_at || +Date.now() < Number(row.original.expires_at)

          return (
            <Text
              color={Color.GREY_500}
              font={{ variation: FontVariation.SMALL_SEMI }}
              lineClamp={1}
              icon="dot"
              iconProps={{ color: isActive ? Color.GREEN_500 : Color.RED_500, size: 20 }}>
              {isActive ? getString('active') : getString('expired')}
            </Text>
          )
        }
      },
      {
        Header: getString('expirationDate'),
        width: '25%',
        Cell: ({ row }: CellProps<TypesToken>) => {
          return (
            <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_500} lineClamp={1}>
              {row.original.expires_at
                ? moment(row.original.expires_at).format('MMM Do, YYYY h:mm:ss a')
                : getString('noExpiration')}
            </Text>
          )
        }
      },
      {
        Header: getString('created'),
        width: '25%',
        Cell: ({ row }: CellProps<TypesToken>) => (
          <Text font={{ variation: FontVariation.SMALL_SEMI }} lineClamp={1}>
            <ReactTimeago date={row.original.issued_at || ''} />
          </Text>
        )
      },
      {
        accessor: 'identifier',
        Header: '',
        width: '5%',
        Cell: ({ row }: CellProps<TypesToken>) => {
          return (
            <OptionsMenuButton
              tooltipProps={{ isDark: true }}
              items={[
                {
                  text: getString('deleteToken'),
                  onClick: () => handleDeleteToken(row.original.identifier as string)
                }
              ]}
            />
          )
        }
      }
    ],
    [userTokens] // eslint-disable-line react-hooks/exhaustive-deps
  )

  const onEditField = async (field: keyof TypesUser, value: string) => {
    try {
      await updateUser({ [field]: value })
      refetchCurrentUser()
    } catch (error) {
      showError(getErrorMessage(error))
    }
  }

  return (
    <Container className={css.mainCtn}>
      <LoadingSpinner visible={currentUserLoading || tokensLoading || sshKeysLoading || systemConfigLoading} />
      <Page.Header title={getString('accountSetting')} />
      <Page.Body>
        <Container className={css.pageCtn}>
          <Card className={css.profileCard}>
            <Avatar
              className={css.avatar}
              name={currentUser?.display_name}
              size="large"
              hoverCard={false}
              color={Color.WHITE}
              backgroundColor={Color.PRIMARY_7}
              borderColor={Color.PRIMARY_8}
            />
            <Container className={css.detailsCtn}>
              <Layout.Horizontal className={css.detailField}>
                <Text width={150} font={{ variation: FontVariation.SMALL }} color={Color.GREY_600}>
                  {getString('userId')}
                </Text>
                <Text font={{ variation: FontVariation.SMALL_SEMI }}>{currentUser?.uid}</Text>
              </Layout.Horizontal>
              <Layout.Horizontal className={css.detailField}>
                <Text width={150} font={{ variation: FontVariation.SMALL }} color={Color.GREY_600}>
                  {getString('accountEmail')}
                </Text>
                <EditableTextField onSave={value => onEditField('email', value)} value={currentUser?.email || ''} />
              </Layout.Horizontal>
              <Layout.Horizontal className={css.detailField}>
                <Text width={150} font={{ variation: FontVariation.SMALL }} color={Color.GREY_600}>
                  {getString('displayName')}
                </Text>
                <EditableTextField
                  onSave={value => onEditField('display_name', value)}
                  value={currentUser?.display_name || ''}
                />
              </Layout.Horizontal>
              <Text
                {...ButtonRoleProps}
                margin={{ top: 'medium' }}
                font={{ variation: FontVariation.TINY }}
                color={Color.PRIMARY_7}
                icon="main-lock"
                iconProps={{ margin: { right: 'xsmall', size: 10 } }}
                onClick={() => history.push(routes.toCODEUserChangePassword())}>
                {getString('changePassword')}
              </Text>
            </Container>
          </Card>
          <Card className={css.logoutCard}>
            <Layout.Horizontal flex>
              <Text
                icon="log-out"
                iconProps={{ margin: { right: 'xsmall' } }}
                font={{ variation: FontVariation.SMALL_SEMI }}>
                {getString('logoutMsg')}
              </Text>
              <Button
                variation={ButtonVariation.SECONDARY}
                size={ButtonSize.SMALL}
                text={getString('logOut')}
                onClick={onLogout}
              />
            </Layout.Horizontal>
          </Card>
          <Container margin={{ top: 'xxxlarge' }}>
            <Container className={css.containerCard}>
              <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
                <Text font={{ variation: FontVariation.CARD_TITLE }}>{getString('sshCard.personalAccessToken')}</Text>
                {isEmpty(userTokens) ? null : (
                  <Button
                    icon="plus"
                    text={getString('newToken.text')}
                    variation={ButtonVariation.LINK}
                    onClick={() => openModal()}
                  />
                )}
              </Layout.Horizontal>
              {isEmpty(userTokens) ? (
                <Container flex={{ justifyContent: 'center' }}>
                  <Layout.Vertical>
                    <Text padding={{ bottom: 'medium' }} font={{ variation: FontVariation.BODY2_SEMI }}>
                      {getString('sshCard.noTokensText')}
                    </Text>
                    <Button
                      icon="plus"
                      text={getString('newToken.text')}
                      variation={ButtonVariation.LINK}
                      onClick={() => openModal()}
                    />
                  </Layout.Vertical>
                </Container>
              ) : (
                <TableV2 data={userTokens || []} columns={columns} />
              )}
            </Container>
          </Container>
          {systemConfig?.ssh_enabled && (
            <Container className={css.containerCard} margin={{ top: 'medium' }}>
              <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
                <Text padding={{ bottom: 'medium' }} font={{ variation: FontVariation.CARD_TITLE }}>
                  {getString('sshCard.mySshKeys')}
                </Text>
                {isEmpty(sshKeys) ? null : (
                  <Button
                    padding={{ bottom: 'medium' }}
                    icon="small-plus"
                    variation={ButtonVariation.LINK}
                    onClick={() => {
                      openSshKeyModal()
                    }}
                    text={getString('sshCard.newSshKey')}
                  />
                )}
              </Layout.Horizontal>
              <Text font={{ variation: FontVariation.BODY2_SEMI }}>{getString('sshCard.sshContent')}</Text>
              {isEmpty(sshKeys) ? (
                <Container flex={{ justifyContent: 'center' }}>
                  <Layout.Vertical>
                    <Text font={{ variation: FontVariation.BODY2_SEMI }} padding={{ top: 'medium', bottom: 'small' }}>
                      {getString('sshCard.noSshKeyText')}
                    </Text>
                    <Button
                      icon="small-plus"
                      variation={ButtonVariation.LINK}
                      onClick={() => {
                        openSshKeyModal()
                      }}
                      text={getString('sshCard.newSshKey')}
                    />
                  </Layout.Vertical>
                </Container>
              ) : (
                <TableV2 columns={sshkeysColumns} data={sshKeys || []} />
              )}
            </Container>
          )}
        </Container>
      </Page.Body>
    </Container>
  )
}

export default UserProfile
