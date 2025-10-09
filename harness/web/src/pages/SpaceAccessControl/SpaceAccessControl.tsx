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
import { Avatar, Button, ButtonVariation, Container, Layout, Page, TableV2, Text, useToaster } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import type { CellProps, Column } from 'react-table'

import { useStrings } from 'framework/strings'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { TypesMembershipUser, useMembershipDelete, useMembershipList } from 'services/code'
import { getErrorMessage, roleStringKeyMap } from 'utils/Utils'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import useAddNewMember from './AddNewMember/AddNewMember'
import css from './SpaceAccessControl.module.scss'

const SpaceAccessControl = () => {
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()
  const space = useGetSpaceParam()

  const { data, refetch, loading } = useMembershipList({
    space_ref: space
  })

  const { openModal } = useAddNewMember({ onClose: refetch })

  const { mutate: deleteMembership } = useMembershipDelete({
    space_ref: space
  })

  const onConfirmAct = useConfirmAct()
  const handleRemoveMember = async (userId: string) =>
    await onConfirmAct({
      action: async () => {
        try {
          await deleteMembership(userId)
          refetch()
          showSuccess(getString('spaceMemberships.removeMembershipToast'))
        } catch (error) {
          showError(getErrorMessage(error))
        }
      },
      message: getString('spaceMemberships.removeMembershipMsg'),
      intent: 'danger',
      title: getString('spaceMemberships.removeMember')
    })

  const columns = useMemo(
    () =>
      [
        {
          Header: getString('user'),
          accessor: 'user',
          sort: true,
          width: '30%',
          Cell: ({ row }: CellProps<TypesMembershipUser>) => (
            <Layout.Horizontal style={{ alignItems: 'center' }}>
              <Avatar
                name={row.original.principal?.display_name}
                size="normal"
                hoverCard={false}
                color={Color.WHITE}
                backgroundColor={Color.PRIMARY_7}
              />
              <Text font={{ variation: FontVariation.SMALL_SEMI }} lineClamp={1}>
                {row.original.principal?.display_name}
              </Text>
            </Layout.Horizontal>
          )
        },
        {
          Header: getString('role'),
          width: '40%',
          accessor: 'role',
          Cell: ({ row }: CellProps<TypesMembershipUser>) => {
            const stringKey = row.original.role ? roleStringKeyMap[row.original.role] : undefined

            return (
              <Text font={{ variation: FontVariation.TINY_SEMI }} color={Color.PRIMARY_9} className={css.roleBadge}>
                {stringKey ? getString(stringKey) : row.original.role}
              </Text>
            )
          }
        },
        {
          Header: getString('email'),
          width: '25%',
          sort: true,
          accessor: 'email',
          Cell: ({ row }: CellProps<TypesMembershipUser>) => (
            <Text font={{ variation: FontVariation.SMALL_SEMI }} lineClamp={1}>
              {row.original.principal?.email}
            </Text>
          )
        },
        {
          accessor: 'actions',
          width: '5%',
          disableSortBy: true,
          Cell: ({ row }: CellProps<TypesMembershipUser>) => {
            return (
              <OptionsMenuButton
                tooltipProps={{ isDark: true }}
                items={[
                  {
                    text: getString('spaceMemberships.removeMember'),
                    onClick: () => handleRemoveMember(row.original.principal?.uid as string)
                  },
                  {
                    text: getString('spaceMemberships.changeRole'),
                    onClick: () => openModal(true, row.original)
                  }
                ]}
              />
            )
          }
        }
      ] as Column<TypesMembershipUser>[],
    [] // eslint-disable-line react-hooks/exhaustive-deps
  )

  return (
    <Container className={css.mainCtn}>
      <Page.Header title={getString('permissionsFor', { name: space })} />
      <Page.Body>
        <Container padding="xlarge">
          <LoadingSpinner visible={loading} />
          <Button
            icon="plus"
            text={getString('addMember')}
            variation={ButtonVariation.PRIMARY}
            margin={{ bottom: 'medium' }}
            onClick={() => openModal()}
          />
          <TableV2 sortable data={data || []} columns={columns} className={css.tableContainer} />
        </Container>
      </Page.Body>
    </Container>
  )
}

export default SpaceAccessControl
