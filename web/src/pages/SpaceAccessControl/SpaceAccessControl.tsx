import React, { useMemo } from 'react'
import { Avatar, Button, ButtonVariation, Container, Layout, Page, TableV2, Text, useToaster } from '@harness/uicore'
import { Color, FontVariation } from '@harness/design-system'
import type { CellProps, Column } from 'react-table'

import { useStrings } from 'framework/strings'
import { useConfirmAct } from 'hooks/useConfirmAction'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import { TypesMembership, useMembershipDelete, useMembershipList } from 'services/code'
import { getErrorMessage } from 'utils/Utils'
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
          width: '30%',
          Cell: ({ row }: CellProps<TypesMembership>) => (
            <Layout.Horizontal style={{ alignItems: 'center' }}>
              <Avatar name={row.original.principal?.display_name} size="normal" hoverCard={false} />
              <Text font={{ variation: FontVariation.SMALL_SEMI }} lineClamp={1}>
                {row.original.principal?.display_name}
              </Text>
            </Layout.Horizontal>
          )
        },
        {
          Header: getString('role'),
          width: '40%',
          Cell: ({ row }: CellProps<TypesMembership>) => (
            <Text font={{ variation: FontVariation.TINY_SEMI }} color={Color.PRIMARY_9} className={css.roleBadge}>
              {row.original.role}
            </Text>
          )
        },
        {
          Header: getString('email'),
          width: '25%',
          Cell: ({ row }: CellProps<TypesMembership>) => (
            <Text font={{ variation: FontVariation.SMALL_SEMI }} lineClamp={1}>
              {row.original.principal?.email}
            </Text>
          )
        },
        {
          accessor: 'action',
          width: '5%',
          Cell: ({ row }: CellProps<TypesMembership>) => {
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
      ] as Column<TypesMembership>[],
    []
  )

  return (
    <Container className={css.mainCtn}>
      <Page.Header title={getString('accessControl')} />
      <Page.Body>
        <Container padding="xlarge">
          <LoadingSpinner visible={loading} />
          <Button
            icon="plus"
            text={getString('newMember')}
            variation={ButtonVariation.PRIMARY}
            margin={{ bottom: 'medium' }}
            onClick={() => openModal()}
          />
          <TableV2 data={data || []} columns={columns} />
        </Container>
      </Page.Body>
    </Container>
  )
}

export default SpaceAccessControl
