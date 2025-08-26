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

import React, { useEffect, useRef, useState } from 'react'
import { Menu, MenuItem, PopoverPosition } from '@blueprintjs/core'
import {
  Button,
  ButtonProps,
  Container,
  Layout,
  ButtonVariation,
  TextInput,
  Text,
  ButtonSize,
  Avatar,
  StringSubstitute
} from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import cx from 'classnames'
import { useGet } from 'restful-react'
import { isEmpty } from 'lodash-es'
import { Icon } from '@harnessio/icons'
import { useParams } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import {
  combineAndNormalizePrincipalsAndGroups,
  getErrorMessage,
  NormalizedPrincipal,
  PrincipalType
} from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { CodeIcon } from 'utils/GitUtils'
import type { TypesPrincipalInfo, TypesPullReq, TypesUserGroupInfo } from 'services/code'
import type { Identifier } from 'utils/types'
import css from './ReviewerSelect.module.scss'

export interface ReviewerSelectProps extends Omit<ButtonProps, 'onSelect'> {
  pullRequestMetadata: TypesPullReq
  onSelect: (principal: NormalizedPrincipal) => void
  standalone?: boolean
}

export const ReviewerSelect: React.FC<ReviewerSelectProps> = ({ pullRequestMetadata, onSelect, ...props }) => {
  const { getString } = useStrings()
  return (
    <Button
      className={css.button}
      text={<span className={css.prefix}>{getString('add')}</span>}
      variation={ButtonVariation.TERTIARY}
      minimal
      size={ButtonSize.SMALL}
      tooltip={<PopoverContent pullRequestMetadata={pullRequestMetadata} onSelect={onSelect} />}
      tooltipProps={{
        interactionKind: 'click',
        usePortal: true,
        position: PopoverPosition.BOTTOM_RIGHT,
        popoverClassName: css.popover
      }}
      tabIndex={0}
      {...props}
    />
  )
}

const PopoverContent: React.FC<ReviewerSelectProps> = ({ pullRequestMetadata, onSelect, standalone }) => {
  const { getString } = useStrings()

  const inputRef = useRef<HTMLInputElement | null>()
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(false)

  return (
    <Container padding="medium" className={css.main}>
      <Layout.Vertical className={css.layout}>
        <TextInput
          className={css.input}
          inputRef={ref => (inputRef.current = ref)}
          defaultValue={query}
          autoFocus
          placeholder={standalone ? getString('findAUser') : getString('findAUserOrUserGroup')}
          onInput={e => {
            const value = (e.currentTarget.value || '').trim()
            setQuery(value)
          }}
          leftIcon={loading ? CodeIcon.InputSpinner : CodeIcon.InputSearch}
          leftIconProps={{
            name: loading ? CodeIcon.InputSpinner : CodeIcon.InputSearch,
            size: 12,
            color: Color.GREY_500
          }}
        />
        <Container className={cx(css.tabContainer)}>
          <ReviewerList
            onSelect={onSelect}
            pullRequestMetadata={pullRequestMetadata}
            query={query}
            setLoading={setLoading}
            standalone={standalone}
          />
        </Container>
      </Layout.Vertical>
    </Container>
  )
}

interface ReviewerListProps extends Omit<ReviewerSelectProps, 'onQuery'> {
  query: string
  setLoading: React.Dispatch<React.SetStateAction<boolean>>
}

function ReviewerList({ pullRequestMetadata, query, onSelect, setLoading, standalone }: ReviewerListProps) {
  const { getString } = useStrings()
  const { routingId } = useAppContext()
  const { accountId, orgIdentifier, projectIdentifier } = useParams<Identifier>()

  const {
    data: principals,
    error: principalsError,
    loading: loadingPrincipals
  } = useGet<TypesPrincipalInfo[]>({
    path: `/api/v1/principals`,
    queryParams: {
      query,
      accountIdentifier: accountId || routingId,
      type: PrincipalType.USER
    }
  })

  const {
    data: userGroups,
    error: userGroupsError,
    loading: loadingUsersGroups
  } = useGet<TypesUserGroupInfo[]>({
    path: `/api/v1/usergroups`,
    queryParams: {
      query,
      accountIdentifier: accountId || routingId,
      orgIdentifier,
      projectIdentifier
    },
    lazy: standalone
  })

  useEffect(() => {
    setLoading(loadingPrincipals || loadingUsersGroups)
  }, [setLoading, loadingPrincipals, loadingUsersGroups])

  const error = principalsError || userGroupsError

  const normalizedPrincipals = combineAndNormalizePrincipalsAndGroups(principals, userGroups)

  return (
    <Container>
      {!!error && (
        <Container flex={{ align: 'center-center' }} padding="large">
          <Text font={{ variation: FontVariation.FORM_MESSAGE_DANGER }}>{getErrorMessage(error)}</Text>
        </Container>
      )}

      {!isEmpty(normalizedPrincipals) ? (
        <Container className={css.listContainer}>
          <Menu>
            {normalizedPrincipals?.map(principal => {
              const { id, email_or_identifier, type, display_name } = principal
              const disabled = id === pullRequestMetadata?.author?.id
              return (
                <MenuItem
                  key={email_or_identifier}
                  className={cx(css.menuItem, { [css.disabled]: disabled })}
                  text={
                    <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'start' }}>
                      {type === PrincipalType.USER ? (
                        <Avatar className={css.avatar} name={display_name} size="small" hoverCard={false} />
                      ) : (
                        <Icon margin={'xsmall'} className={cx(css.avatar, css.ugicon)} name="user-groups" size={18} />
                      )}
                      <Layout.Vertical
                        padding={{ left: 'small' }}
                        style={type === PrincipalType.USER_GROUP ? { marginLeft: '2px' } : undefined}>
                        <Text>
                          <strong>{display_name}</strong>
                        </Text>
                        <Text>{email_or_identifier}</Text>
                      </Layout.Vertical>
                    </Layout.Horizontal>
                  }
                  labelElement={disabled ? <Text className={css.owner}>owner</Text> : undefined}
                  disabled={disabled}
                  onClick={() => onSelect(principal)}
                />
              )
            })}
          </Menu>
        </Container>
      ) : (
        <Container className={css.noTextContainer} flex={{ align: 'center-center' }} padding="large">
          {
            <Text className={css.noWrapText} flex padding={{ top: 'small' }}>
              <StringSubstitute
                str={getString('noUsersFound')}
                vars={{
                  reviewer: (
                    <Text
                      padding={{ right: 'tiny' }}
                      tooltipProps={{ popoverClassName: css.reviewerPopover }}
                      className={css.noReviewerContainer}
                      lineClamp={1}>
                      <strong> {query}</strong>
                    </Text>
                  )
                }}
              />
            </Text>
          }
        </Container>
      )}
    </Container>
  )
}
