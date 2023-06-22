/*
 * Copyright 2021 Harness Inc. All rights reserved.
 * Use of this source code is governed by the PolyForm Shield 1.0.0 license
 * that can be found in the licenses directory at the root of this repository, also available at
 * https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
 */

import React, { useEffect, useRef, useState } from 'react'
import { Icon as BPIcon, Menu, MenuItem, PopoverPosition } from '@blueprintjs/core'
import {
  Button,
  ButtonProps,
  Container,
  Layout,
  ButtonVariation,
  TextInput,
  FontVariation,
  Text,
  ButtonSize,
  Avatar
} from '@harness/uicore'
import cx from 'classnames'
import { useGet } from 'restful-react'
import { String, useStrings } from 'framework/strings'
import { getErrorMessage, LIST_FETCHING_LIMIT } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { CodeIcon, GitInfoProps, REFS_TAGS_PREFIX } from 'utils/GitUtils'
import { usePageIndex } from 'hooks/usePageIndex'
import css from './ReviewerSelect.module.scss'

export interface ReviewerSelectProps extends Omit<ButtonProps, 'onSelect'>, Pick<GitInfoProps, 'repoMetadata'> {
  gitRef: string
  onSelect: (id: number) => void
  labelPrefix?: string
  placeHolder?: string
}

export const ReviewerSelect: React.FC<ReviewerSelectProps> = ({
  repoMetadata,
  gitRef,
  onSelect,
  labelPrefix,
  placeHolder,
  ...props
}) => {
  const text = gitRef.replace(REFS_TAGS_PREFIX, '')

  return (
    <Button
      className={css.button}
      text={
        text ? (
          labelPrefix ? (
            <>
              <span className={css.prefix}>{labelPrefix}</span>
              {text}
            </>
          ) : (
            text
          )
        ) : (
          <span className={css.prefix}>{placeHolder}</span>
        )
      }
      variation={ButtonVariation.TERTIARY}
      minimal
      size={ButtonSize.SMALL}
      tooltip={
        <PopoverContent
          gitRef={gitRef}
          repoMetadata={repoMetadata}
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

const PopoverContent: React.FC<ReviewerSelectProps> = ({ repoMetadata, gitRef, onSelect }) => {
  const { getString } = useStrings()

  const inputRef = useRef<HTMLInputElement | null>()
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(false)

  return (
    <Container padding="medium" className={css.main}>
      <Layout.Vertical spacing="small">
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
        />

        <Container className={cx(css.tabContainer)}>
          <GitRefList
            gitRef={gitRef}
            onSelect={display_name => onSelect(display_name)}
            repoMetadata={repoMetadata}
            query={query}
            setLoading={setLoading}
          />
        </Container>
      </Layout.Vertical>
    </Container>
  )
}

interface GitRefListProps extends Omit<ReviewerSelectProps, 'onQuery'> {
  query: string
  setLoading: React.Dispatch<React.SetStateAction<boolean>>
}

function GitRefList({
  gitRef,
  query,
  onSelect,

  setLoading
}: GitRefListProps) {
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
        <Container className={css.listContainer} padding={{ top: 'small', bottom: 'small' }}>
          <Menu>
            {data.map(({ display_name, email, id }) => (
              <MenuItem
                key={email}
                text={
                  <Layout.Horizontal>
                    <Avatar name={display_name} size="small" hoverCard={false} />

                    <Layout.Vertical padding={{ left: 'small' }}>
                      <Text>
                        <strong>{display_name}</strong>
                      </Text>
                      <Text>{email}</Text>
                    </Layout.Vertical>
                  </Layout.Horizontal>
                }
                labelElement={display_name === gitRef ? <BPIcon icon="small-tick" /> : undefined}
                disabled={display_name === gitRef}
                onClick={() => onSelect(id as number)}
              />
            ))}
          </Menu>
        </Container>
      )}

      {data?.length === 0 && (
        <Container flex={{ align: 'center-center' }} padding="large">
          {
            <Text padding={{ top: 'small' }}>
              <String stringID="reviewerNotFound" tagName="span" vars={{ reviewer: query }} useRichText />
            </Text>
          }
        </Container>
      )}
    </Container>
  )
}
