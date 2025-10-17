import React, { useMemo } from 'react'
import { PopoverInteractionKind, Position } from '@blueprintjs/core'
import { Avatar, Container, Layout, Popover, Text } from '@harnessio/uicore'
import { isEmpty } from 'lodash-es'
import { Icon } from '@harnessio/icons'
import { Color } from '@harnessio/design-system'
import type { TypesPrincipalInfo, TypesUserGroupInfo } from 'services/code'
import { combineAndNormalizePrincipalsAndGroups, PrincipalType } from 'utils/Utils'
import css from '../PullRequestOverviewPanel.module.scss'

const ReviewersPanel = ({
  principals,
  userGroups
}: {
  principals: TypesPrincipalInfo[]
  userGroups?: TypesUserGroupInfo[]
}) => {
  const normalizedPrincipals = useMemo(
    () => combineAndNormalizePrincipalsAndGroups(principals, userGroups, true),
    [principals, userGroups]
  )

  if (isEmpty(normalizedPrincipals)) {
    return null
  }

  return (
    <Layout.Horizontal className={css.reviewerContainer} spacing="tiny">
      {normalizedPrincipals?.slice(0, 3).map(({ display_name, email_or_identifier, type }) => {
        if (type === PrincipalType.USER_GROUP) {
          return (
            <Popover
              key={email_or_identifier}
              interactionKind={PopoverInteractionKind.HOVER}
              position={Position.TOP}
              content={<Container padding={'small'}>{display_name}</Container>}>
              <Icon margin={'xsmall'} className={css.ugicon} name="user-groups" size={18} />
            </Popover>
          )
        }
        return (
          <Avatar
            key={email_or_identifier}
            hoverCard
            email={email_or_identifier || ''}
            size="small"
            name={display_name || ''}
          />
        )
      })}
      {normalizedPrincipals?.length > 3 && (
        <Text
          tooltipProps={{ isDark: true }}
          tooltip={
            <Container width={215} padding={'small'}>
              <Layout.Horizontal className={css.reviewerTooltip}>
                {normalizedPrincipals?.map(({ display_name, id }, idx) => (
                  <Text
                    key={`text-${id}-${display_name}`}
                    lineClamp={1}
                    color={Color.GREY_0}
                    padding={{ right: 'small' }}>
                    {`${display_name}${normalizedPrincipals?.length === idx + 1 ? '' : ', '}`}
                  </Text>
                ))}
              </Layout.Horizontal>
            </Container>
          }
          flex={{ alignItems: 'center' }}>{`+${normalizedPrincipals?.length - 3}`}</Text>
      )}
    </Layout.Horizontal>
  )
}

export default ReviewersPanel
