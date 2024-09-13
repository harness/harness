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
import { useStrings } from 'framework/strings'
import { getErrorMessage, LIST_FETCHING_LIMIT } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { CodeIcon } from 'utils/GitUtils'
import { usePageIndex } from 'hooks/usePageIndex'
import type { TypesPullReq } from 'services/code'
import css from './ReviewerSelect.module.scss'

export interface ReviewerSelectProps extends Omit<ButtonProps, 'onSelect'> {
  pullRequestMetadata: TypesPullReq
  onSelect: (id: number) => void
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
      tooltip={
        <PopoverContent
          pullRequestMetadata={pullRequestMetadata}
          onSelect={ref => {
            onSelect(ref)
          }}
        />
      }
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

const PopoverContent: React.FC<ReviewerSelectProps> = ({ pullRequestMetadata, onSelect }) => {
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
          placeholder={getString('findAUser')}
          onInput={e => {
            const _value = (e.currentTarget.value || '').trim()
            setQuery(_value)
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
            onSelect={display_name => onSelect(display_name)}
            pullRequestMetadata={pullRequestMetadata}
            query={query}
            setLoading={setLoading}
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

function ReviewerList({
  pullRequestMetadata,
  query,
  onSelect,

  setLoading
}: ReviewerListProps) {
  const { getString } = useStrings()
  const [page] = usePageIndex(1)
  const { routingId } = useAppContext()
  const { data, error, loading } = useGet<Unknown[]>({
    path: `/api/v1/principals`,
    queryParams: {
      query: query,
      limit: LIST_FETCHING_LIMIT,
      page: page,
      accountIdentifier: routingId,
      type: 'user'
    }
  })

  useEffect(() => {
    setLoading(loading)
  }, [setLoading, loading])

  return (
    <Container>
      {!!error && (
        <Container flex={{ align: 'center-center' }} padding="large">
          {!!error && <Text font={{ variation: FontVariation.FORM_MESSAGE_DANGER }}>{getErrorMessage(error)}</Text>}
        </Container>
      )}

      {!!data?.length && (
        <Container className={css.listContainer}>
          <Menu>
            {data.map(({ display_name, email, id }) => {
              const disabled = id === pullRequestMetadata?.author?.id
              return (
                <MenuItem
                  key={email}
                  className={cx(css.menuItem, { [css.disabled]: disabled })}
                  text={
                    <Layout.Horizontal>
                      <Avatar className={css.avatar} name={display_name} size="small" hoverCard={false} />

                      <Layout.Vertical padding={{ left: 'small' }}>
                        <Text>
                          <strong>{display_name}</strong>
                        </Text>
                        <Text>{email}</Text>
                      </Layout.Vertical>
                    </Layout.Horizontal>
                  }
                  labelElement={disabled ? <Text className={css.owner}>owner</Text> : undefined}
                  disabled={disabled}
                  onClick={() => onSelect(id as number)}
                />
              )
            })}
          </Menu>
        </Container>
      )}

      {data?.length === 0 && (
        <Container className={css.noTextContainer} flex={{ align: 'center-center' }} padding="large">
          {
            <Text className={css.noWrapText} flex padding={{ top: 'small' }}>
              <StringSubstitute
                str={getString('reviewerNotFound')}
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
