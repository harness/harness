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

import React, { useCallback, useEffect, useRef, useState } from 'react'
import type { EditorView } from '@codemirror/view'
import { Render, Match, Truthy, Falsy, Else } from 'react-jsx-match'
import { Container, Layout, Avatar, TextInput, Text, FlexExpander, Button, useIsMounted } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import cx from 'classnames'
import { isEqual, noop, defaultTo } from 'lodash-es'
import { TimePopoverWithLocal } from 'utils/timePopoverLocal/TimePopoverWithLocal'
import { useStrings } from 'framework/strings'
import { ThreadSection } from 'components/ThreadSection/ThreadSection'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { useAppContext } from 'AppContext'
import type { TypesRepository } from 'services/code'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { MarkdownEditorWithPreview } from 'components/MarkdownEditorWithPreview/MarkdownEditorWithPreview'
import { MarkdownViewer } from 'components/MarkdownViewer/MarkdownViewer'
import { ButtonRoleProps } from 'utils/Utils'
import { useResizeObserver } from 'hooks/useResizeObserver'
import { useCustomEventListener } from 'hooks/useEventListener'
import css from './CommentBox.module.scss'

export interface CommentItem<T = unknown> {
  id: number
  author: string
  created: string | number
  edited: string | number
  updated: string | number
  deleted: string | number
  outdated: boolean
  content: string
  payload?: T // optional payload for callers to handle on callback calls
}

export enum CommentAction {
  NEW = 'new',
  UPDATE = 'update',
  REPLY = 'reply',
  DELETE = 'delete',
  RESOLVE = 'resolve',
  REACTIVATE = 'reactivate'
}

// Outlets are used to insert additional components into CommentBox
export enum CommentBoxOutletPosition {
  TOP = 'top',
  BOTTOM = 'bottom',
  TOP_OF_FIRST_COMMENT = 'top_of_first_comment',
  BOTTOM_OF_COMMENT_EDITOR = 'bottom_of_comment_editor',
  LEFT_OF_OPTIONS_MENU = 'left_of_options_menu',
  RIGHT_OF_REPLY_PLACEHOLDER = 'right_of_reply_placeholder',
  BETWEEN_SAVE_AND_CANCEL_BUTTONS = 'between_save_and_cancel_buttons'
}

export type CommentItemsHandler<T> = (t: T) => void

interface CommentBoxProps<T> {
  outerClassName?: string
  editorClassName?: string
  boxClassName?: string
  onHeightChange?: (height: number) => void
  initialContent?: string
  width?: string
  fluid?: boolean
  resetOnSave?: boolean
  hideCancel?: boolean
  currentUserName: string
  commentItems: CommentItem<T>[]
  handleAction: (
    action: CommentAction,
    content: string,
    atCommentItem?: CommentItem<T>
  ) => Promise<[boolean, CommentItem<T> | undefined]>
  onCancel?: () => void
  setDirty: React.Dispatch<React.SetStateAction<boolean>>
  outlets?: Partial<Record<CommentBoxOutletPosition, React.ReactNode>>
  autoFocusAndPosition?: boolean
  enableReplyPlaceHolder?: boolean
  repoMetadata: TypesRepository | undefined
  standalone: boolean
  routingId: string
}

const CommentBoxInternal = <T = unknown,>({
  outerClassName,
  editorClassName,
  boxClassName,
  onHeightChange = noop,
  initialContent = '',
  width,
  fluid,
  commentItems = [],
  currentUserName,
  handleAction,
  onCancel = noop,
  hideCancel,
  resetOnSave,
  setDirty,
  outlets = {},
  autoFocusAndPosition,
  enableReplyPlaceHolder,
  repoMetadata,
  standalone,
  routingId
}: CommentBoxProps<T>) => {
  const { getString } = useStrings()
  const [comments, setComments] = useState<CommentItem<T>[]>(commentItems)
  const enableReplyPlaceHolderRef = useRef<boolean | undefined>(enableReplyPlaceHolder)
  const [showReplyPlaceHolder, setShowReplyPlaceHolder] = useState(enableReplyPlaceHolder)
  const [markdown, setMarkdown] = useState(initialContent)
  const [dirties, setDirties] = useState<Record<string, boolean>>({})
  const containerRef = useRef<HTMLDivElement>(null)
  const isMounted = useIsMounted()

  useResizeObserver(
    containerRef,
    useCallback(
      dom => {
        if (isMounted.current && dom) onHeightChange(dom.offsetHeight)
      },
      [onHeightChange, isMounted]
    )
  )

  useCustomEventListener(
    customEventForCommentWithId(comments?.[0]?.id),
    useCallback((event: CustomEvent) => {
      const updatedComments = event.detail
      setComments(_comments => (isEqual(_comments, updatedComments) ? _comments : updatedComments))
    }, []),
    () => !!comments?.[0]?.id
  )

  const _onCancel = useCallback(() => {
    setMarkdown('')
    setShowReplyPlaceHolder(true)
    if (onCancel && !comments.length) {
      onCancel()
    }
  }, [setShowReplyPlaceHolder, onCancel, comments.length])
  const hidePlaceHolder = useCallback(() => setShowReplyPlaceHolder(false), [setShowReplyPlaceHolder])
  const onQuote = useCallback((content: string) => {
    const replyContent = content
      .split(CRLF)
      .map(line => `> ${line}`)
      .concat([CRLF])
      .join(CRLF)
    setShowReplyPlaceHolder(false)
    setMarkdown(replyContent)
  }, [])
  const viewRef = useRef<EditorView>()

  useEffect(() => {
    setDirty(_oldDirty => {
      const someDirty = Object.values(dirties).some(dirty => dirty)
      return someDirty !== _oldDirty ? someDirty : _oldDirty
    })
  }, [dirties, setDirty])

  return (
    <Container
      className={cx(css.main, { [css.fluid]: fluid }, outerClassName)}
      padding={!fluid ? 'medium' : undefined}
      width={width}
      ref={containerRef}
      data-comment-thread-id={comments?.[0]?.id || ''}>
      <Container className={cx(boxClassName, css.box)}>
        {outlets[CommentBoxOutletPosition.TOP]}

        <Layout.Vertical>
          {/* CommentsThread is rendered only when comments.length > 0 */}
          <CommentsThread<T>
            repoMetadata={repoMetadata}
            commentItems={comments}
            onQuote={onQuote}
            handleAction={async (action, content, atCommentItem) => {
              const [result, updatedItem] = await handleAction(action, content, atCommentItem)

              if (result && action === CommentAction.DELETE && atCommentItem) {
                atCommentItem.edited = atCommentItem.deleted = Date.now()
                setComments([...comments])
              }

              return [result, updatedItem]
            }}
            setDirty={(index, dirty) => {
              setDirties({ ...dirties, [index]: dirty })
            }}
            outlets={outlets}
          />
          <Match expr={showReplyPlaceHolder && enableReplyPlaceHolderRef.current}>
            <Truthy>
              <Container>
                <Layout.Horizontal
                  spacing="small"
                  className={cx(css.replyPlaceHolder, editorClassName)}
                  padding="medium">
                  <Avatar name={currentUserName} size="small" hoverCard={false} />
                  <TextInput
                    {...ButtonRoleProps}
                    placeholder={getString(comments.length ? 'replyHere' : 'leaveAComment')}
                    onFocus={hidePlaceHolder}
                    onClick={hidePlaceHolder}
                  />
                  {outlets[CommentBoxOutletPosition.RIGHT_OF_REPLY_PLACEHOLDER]}
                </Layout.Horizontal>
              </Container>
            </Truthy>
            <Falsy>
              <Container className={cx(css.newCommentContainer, { [css.hasThread]: !!comments.length })}>
                <MarkdownEditorWithPreview
                  routingId={routingId}
                  standalone={standalone}
                  repoMetadata={repoMetadata}
                  className={editorClassName}
                  viewRef={viewRef}
                  noBorder
                  i18n={{
                    placeHolder: getString(comments.length ? 'replyHere' : 'leaveAComment'),
                    tabEdit: getString('write'),
                    tabPreview: getString('preview'),
                    save: getString(comments.length ? 'reply' : 'comment'),
                    cancel: getString('cancel')
                  }}
                  value={markdown}
                  onChange={setMarkdown}
                  onSave={async (value: string) => {
                    if (handleAction) {
                      const [result, updatedItem] = await handleAction(
                        comments.length ? CommentAction.REPLY : CommentAction.NEW,
                        value,
                        comments[0]
                      )

                      if (result) {
                        setMarkdown('')
                        setShowReplyPlaceHolder(true)

                        // New comment? Enable the reply place-holder after saving
                        if (!comments.length) {
                          enableReplyPlaceHolderRef.current = true
                        }

                        if (resetOnSave) {
                          viewRef.current?.dispatch({
                            changes: {
                              from: 0,
                              to: viewRef.current.state.doc.length,
                              insert: ''
                            }
                          })
                        } else {
                          setComments([...comments, updatedItem as CommentItem<T>])
                        }
                      }
                    }
                  }}
                  secondarySaveButton={
                    comments.length
                      ? (outlets[CommentBoxOutletPosition.BETWEEN_SAVE_AND_CANCEL_BUTTONS] as typeof Button)
                      : undefined
                  }
                  onCancel={_onCancel}
                  hideCancel={hideCancel}
                  setDirty={_dirty => {
                    setDirties({ ...dirties, ['new']: _dirty })
                  }}
                  autoFocusAndPosition={autoFocusAndPosition ? !showReplyPlaceHolder : false}
                />
              </Container>
            </Falsy>
          </Match>
        </Layout.Vertical>
      </Container>
    </Container>
  )
}

interface CommentsThreadProps<T> extends Pick<CommentBoxProps<T>, 'commentItems' | 'handleAction' | 'outlets'> {
  onQuote: (content: string) => void
  setDirty: (index: number, dirty: boolean) => void
  repoMetadata: TypesRepository | undefined
}

const CommentsThread = <T = unknown,>({
  onQuote,
  commentItems = [],
  handleAction,
  setDirty,
  outlets = {},
  repoMetadata
}: CommentsThreadProps<T>) => {
  const { getString } = useStrings()
  const { standalone, routingId } = useAppContext()
  const [editIndexes, setEditIndexes] = useState<Record<number, boolean>>({})
  const resetStateAtIndex = useCallback(
    (index: number) => {
      delete editIndexes[index]
      setEditIndexes({ ...editIndexes })
    },
    [editIndexes]
  )

  return (
    <Render when={commentItems.length}>
      <Container className={css.viewer} padding="xlarge">
        {commentItems.map((commentItem, index) => {
          const isLastItem = index === commentItems.length - 1

          return (
            <ThreadSection
              key={index}
              title={
                <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }}>
                  <Text inline icon="code-chat"></Text>
                  <Avatar name={commentItem?.author} size="small" hoverCard={false} />
                  <Text inline>
                    <strong>{commentItem?.author}</strong>
                  </Text>
                  <PipeSeparator height={8} />
                  <Text inline font={{ variation: FontVariation.SMALL }} color={Color.GREY_400}>
                    <TimePopoverWithLocal
                      time={defaultTo(commentItem?.edited as number, 0)}
                      inline={false}
                      font={{ variation: FontVariation.SMALL }}
                      color={Color.GREY_400}
                    />
                  </Text>

                  <Render when={commentItem?.edited !== commentItem?.created || !!commentItem?.deleted}>
                    <>
                      <PipeSeparator height={8} />
                      <Text inline font={{ variation: FontVariation.SMALL }} color={Color.GREY_400}>
                        {getString(commentItem?.deleted ? 'deleted' : 'edited')}
                      </Text>
                    </>
                  </Render>

                  <Render when={commentItem?.outdated}>
                    <Text inline font={{ variation: FontVariation.SMALL }} className={css.outdated}>
                      {getString('pr.outdated')}
                    </Text>
                  </Render>

                  <FlexExpander />
                  <Layout.Horizontal>
                    <Render when={index === 0 && outlets[CommentBoxOutletPosition.LEFT_OF_OPTIONS_MENU]}>
                      <Container padding={{ right: 'medium' }}>
                        {outlets[CommentBoxOutletPosition.LEFT_OF_OPTIONS_MENU]}
                      </Container>
                    </Render>
                    <Render when={!commentItem?.deleted}>
                      <OptionsMenuButton
                        isDark={true}
                        icon="Options"
                        iconProps={{ size: 14 }}
                        style={{ padding: '5px' }}
                        width="100px"
                        items={[
                          {
                            hasIcon: true,
                            className: css.optionMenuIcon,
                            iconName: 'Edit',
                            text: getString('edit'),
                            onClick: () => setEditIndexes({ ...editIndexes, ...{ [index]: true } })
                          },
                          {
                            hasIcon: true,
                            className: css.optionMenuIcon,
                            iconName: 'code-quote',
                            text: getString('quote'),
                            onClick: () => onQuote(commentItem?.content)
                          },
                          '-',
                          {
                            className: css.deleteIcon,
                            hasIcon: true,
                            iconName: 'main-trash',
                            isDanger: true,
                            text: getString('delete'),
                            onClick: async () => {
                              if (await handleAction(CommentAction.DELETE, '', commentItem)) {
                                resetStateAtIndex(index)
                              }
                            }
                          }
                        ]}
                      />
                    </Render>
                  </Layout.Horizontal>
                </Layout.Horizontal>
              }
              hideGutter={isLastItem}>
              <Container padding={{ bottom: isLastItem ? undefined : 'xsmall' }} data-comment-id={commentItem.id}>
                <Render when={index === 0 && outlets[CommentBoxOutletPosition.TOP_OF_FIRST_COMMENT]}>
                  <Container className={cx(css.outletTopOfFirstOfComment, { [css.standalone]: standalone })}>
                    {outlets[CommentBoxOutletPosition.TOP_OF_FIRST_COMMENT]}
                  </Container>
                </Render>

                <Match expr={editIndexes[index]}>
                  <Truthy>
                    <Container className={css.editCommentContainer}>
                      <MarkdownEditorWithPreview
                        routingId={routingId}
                        standalone={standalone}
                        repoMetadata={repoMetadata}
                        value={commentItem?.content}
                        onSave={async value => {
                          if (await handleAction(CommentAction.UPDATE, value, commentItem)) {
                            commentItem.content = value
                            resetStateAtIndex(index)
                          }
                        }}
                        onCancel={() => resetStateAtIndex(index)}
                        setDirty={_dirty => {
                          setDirty(index, _dirty)
                        }}
                        i18n={{
                          placeHolder: getString('leaveAComment'),
                          tabEdit: getString('write'),
                          tabPreview: getString('preview'),
                          save: getString('save'),
                          cancel: getString('cancel')
                        }}
                        autoFocusAndPosition
                      />
                    </Container>
                  </Truthy>
                  <Else>
                    <Match expr={commentItem?.deleted}>
                      <Truthy>
                        <Text className={css.deleted}>{getString('commentDeleted')}</Text>
                      </Truthy>
                      <Else>
                        <MarkdownViewer source={commentItem?.content} />
                      </Else>
                    </Match>
                  </Else>
                </Match>
              </Container>
            </ThreadSection>
          )
        })}
      </Container>
    </Render>
  )
}

export const CommentBox = React.memo(CommentBoxInternal)

export const customEventForCommentWithId = (id: number) => `CommentBoxCustomEvent-${id}`

const CRLF = '\n'
