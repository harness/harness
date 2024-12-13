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

import React, { createRef, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { EditorView } from '@codemirror/view'
import { Render, Match, Truthy, Falsy, Else } from 'react-jsx-match'
import { Container, Layout, Avatar, TextInput, Text, FlexExpander, Button, useIsMounted } from '@harnessio/uicore'
import { Color, FontVariation } from '@harnessio/design-system'
import cx from 'classnames'
import { isEqual, noop, defaultTo, get } from 'lodash-es'
import { TimePopoverWithLocal } from 'utils/timePopoverLocal/TimePopoverWithLocal'
import { useStrings } from 'framework/strings'
import { ThreadSection } from 'components/ThreadSection/ThreadSection'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { useAppContext } from 'AppContext'
import type { RepoRepositoryOutput, TypesPrincipalInfo, TypesPullReqActivity } from 'services/code'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { MarkdownEditorWithPreview } from 'components/MarkdownEditorWithPreview/MarkdownEditorWithPreview'
import { MarkdownViewer } from 'components/MarkdownViewer/MarkdownViewer'
import { ButtonRoleProps, CodeCommentState, replaceMentionEmailWithId, replaceMentionIdWithEmail } from 'utils/Utils'
import { useResizeObserver } from 'hooks/useResizeObserver'
import { PR_COMMENT_STATUS_CHANGED_EVENT } from 'hooks/useEmitCodeCommentStatus'
import { useCustomEventListener } from 'hooks/useEventListener'
import type { SuggestionBlock } from 'components/SuggestionBlock/SuggestionBlock'
import type { CommentRestorationTrackingState, DiffViewerExchangeState } from 'components/DiffViewer/DiffViewer'
import commentActiveIconUrl from './comment.svg?url'
import commentResolvedIconUrl from './comment-resolved.svg?url'
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
  commentThreadId?: number
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
  repoMetadata: RepoRepositoryOutput | undefined
  standalone: boolean
  routingId: string
  copyLinkToComment: (commentId: number, commentItem: CommentItem<T>) => void
  suggestionBlock?: SuggestionBlock
  memorizedState?: CommentRestorationTrackingState
  commentsVisibilityAtLineNumber?: DiffViewerExchangeState['commentsVisibilityAtLineNumber']
}

const CommentBoxInternal = <T = unknown,>({
  outerClassName,
  editorClassName,
  boxClassName,
  onHeightChange = noop,
  initialContent = '',
  width,
  fluid,
  commentThreadId,
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
  routingId,
  copyLinkToComment,
  suggestionBlock,
  memorizedState,
  commentsVisibilityAtLineNumber
}: CommentBoxProps<T>) => {
  const { getString } = useStrings()
  const [comments, setComments] = useState<CommentItem<T>[]>(commentItems)
  const enableReplyPlaceHolderRef = useRef<boolean | undefined>(enableReplyPlaceHolder)
  const [showReplyPlaceHolder, setShowReplyPlaceHolder] = useState(enableReplyPlaceHolder)
  const [markdown, setMarkdown] = useState(initialContent)
  const [dirties, setDirties] = useState<Record<string, boolean>>({})
  const containerRef = useRef<HTMLDivElement>(null)
  const isMounted = useIsMounted()
  const [mentionsMap, setMentionsMap] = useState<{ [key: string]: TypesPrincipalInfo }>({})
  const clearMemorizedState = useCallback(() => {
    if (memorizedState) {
      delete memorizedState.showReplyPlaceHolder
      delete memorizedState.uncommittedText
    }
  }, [memorizedState])

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

    clearMemorizedState()

    if (onCancel && !comments.length) {
      onCancel()
    }
  }, [setShowReplyPlaceHolder, onCancel, comments.length, clearMemorizedState])
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

  useEffect(
    // This function restores CommentBox internal states from memorizedState
    // after it got destroyed during HTML/textContent serialization/deserialization
    // This approach is not optimized, we probably have to think about a shared
    // store per diff or something else to make the flow nicer
    function serializeNewCommentInfo() {
      if (!commentThreadId || !memorizedState) return

      if (commentThreadId < 0) {
        if (!comments?.[0]?.id) {
          if (!markdown && memorizedState.uncommittedText) {
            setMarkdown(memorizedState.uncommittedText)
            viewRef.current?.dispatch({
              changes: {
                from: 0,
                to: viewRef.current.state.doc.length,
                insert: memorizedState.uncommittedText
              }
            })
            viewRef.current?.contentDOM?.blur()
          } else {
            memorizedState.uncommittedText = markdown
            memorizedState.showReplyPlaceHolder = showReplyPlaceHolder
          }
        } else {
          clearMemorizedState()
        }
      } else if (commentThreadId > 0) {
        if (!showReplyPlaceHolder) {
          if (markdown) {
            memorizedState.uncommittedText = markdown
            memorizedState.showReplyPlaceHolder = false
          }
        } else {
          if (!markdown && memorizedState.showReplyPlaceHolder === false) {
            setShowReplyPlaceHolder(false)

            const { uncommittedText = '' } = memorizedState

            setTimeout(() => {
              setMarkdown(uncommittedText)
              viewRef.current?.dispatch({
                changes: {
                  from: 0,
                  to: viewRef.current.state.doc.length,
                  insert: uncommittedText
                }
              })
              viewRef.current?.contentDOM?.blur()
            }, 0)
          }

          delete memorizedState.showReplyPlaceHolder
          delete memorizedState.uncommittedText
        }
      }
    },
    [markdown, commentThreadId, comments, memorizedState, clearMemorizedState, showReplyPlaceHolder]
  )

  return (
    <Container
      className={cx(css.main, { [css.fluid]: fluid }, outerClassName)}
      padding={!fluid ? 'medium' : undefined}
      width={width}
      ref={containerRef}
      data-comment-thread-id={comments?.[0]?.id || commentThreadId || ''}>
      {outlets[CommentBoxOutletPosition.TOP]}
      <Container className={cx(boxClassName, css.box)}>
        <Layout.Vertical>
          {/* CommentsThread is rendered only when comments.length > 0 */}
          <CommentsThread<T>
            repoMetadata={repoMetadata}
            commentItems={comments}
            setMentionsMap={setMentionsMap}
            mentionsMap={mentionsMap}
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
            copyLinkToComment={copyLinkToComment}
            suggestionBlock={suggestionBlock}
            memorizedState={memorizedState}
            commentsVisibilityAtLineNumber={commentsVisibilityAtLineNumber}
          />
          <Match expr={showReplyPlaceHolder && enableReplyPlaceHolderRef.current}>
            <Truthy>
              <Container data-reply-placeholder>
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
              <Container
                className={cx(css.newCommentContainer, { [css.hasThread]: !!comments.length })}
                data-comment-editor-shown="true">
                <MarkdownEditorWithPreview
                  routingId={routingId}
                  standalone={standalone}
                  repoMetadata={repoMetadata}
                  className={editorClassName}
                  viewRef={viewRef}
                  setMentionsMap={setMentionsMap}
                  mentionsMap={mentionsMap}
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
                    clearMemorizedState()

                    if (handleAction) {
                      const [result, updatedItem] = await handleAction(
                        comments.length ? CommentAction.REPLY : CommentAction.NEW,
                        replaceMentionEmailWithId(value, mentionsMap),
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
                          if (
                            updatedItem?.content &&
                            (updatedItem as CommentItem<TypesPullReqActivity>)?.payload?.mentions
                          ) {
                            updatedItem.content = replaceMentionIdWithEmail(
                              updatedItem?.content,
                              (updatedItem as CommentItem<TypesPullReqActivity>)?.payload?.mentions ?? {}
                            )
                          }

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
                  suggestionBlock={suggestionBlock}
                />
              </Container>
            </Falsy>
          </Match>
        </Layout.Vertical>
      </Container>
      {outlets[CommentBoxOutletPosition.BOTTOM]}
    </Container>
  )
}

interface CommentsThreadProps<T>
  extends Pick<
    CommentBoxProps<T>,
    | 'commentItems'
    | 'handleAction'
    | 'outlets'
    | 'copyLinkToComment'
    | 'suggestionBlock'
    | 'memorizedState'
    | 'commentsVisibilityAtLineNumber'
  > {
  onQuote: (content: string) => void
  setDirty: (index: number, dirty: boolean) => void
  repoMetadata: RepoRepositoryOutput | undefined
  setMentionsMap?: React.Dispatch<
    React.SetStateAction<{
      [key: string]: TypesPrincipalInfo
    }>
  >
  mentionsMap?: {
    [key: string]: TypesPrincipalInfo
  }
}

const CommentsThread = <T = unknown,>({
  onQuote,
  commentItems = [],
  handleAction,
  setDirty,
  outlets = {},
  repoMetadata,
  copyLinkToComment,
  suggestionBlock,
  memorizedState,
  commentsVisibilityAtLineNumber,
  setMentionsMap,
  mentionsMap
}: CommentsThreadProps<T>) => {
  const { getString } = useStrings()
  const { standalone, routingId } = useAppContext()
  const [editIndexes, setEditIndexes] = useState<Record<number, boolean>>({})
  const resetStateAtIndex = useCallback(
    (index: number, commentItem: CommentItem<T>) => {
      delete editIndexes[index]
      setEditIndexes({ ...editIndexes })

      if (memorizedState?.uncommittedEditComments && commentItem?.id) {
        memorizedState.uncommittedEditComments.delete(commentItem.id)
      }
    },
    [editIndexes, memorizedState]
  )
  const isCommentThreadResolved = useMemo(() => !!get(commentItems[0], 'payload.resolved'), [commentItems])
  const domRef = useRef<HTMLElement>()
  const internalFlags = useRef({ initialized: false })

  const handleCommentStatusChange = useCallback((e: any, toggleComments?: (e: KeyboardEvent | MouseEvent) => void) => {
    if (commentItems[0].id === e.detail.id && e.detail.status === CodeCommentState.RESOLVED) {
      toggleComments?.(e)
    }
  }, [])

  useEffect(
    function renderToggleCommentsButton() {
      // Get the row that contains the comment. If the comment is spanned for multiple lines, the
      // row is the last row
      let annotatedRow = domRef.current?.closest('tr') as HTMLTableRowElement

      // Make sure annotatedRow is not one of rows which renders a comment
      while (annotatedRow && !annotatedRow.dataset.sourceLineNumber) {
        annotatedRow = annotatedRow?.previousElementSibling as HTMLTableRowElement
      }

      if (annotatedRow) {
        const lineNumColDOM = annotatedRow.firstElementChild as HTMLElement
        const sourceLineNumber = annotatedRow.dataset.sourceLineNumber
        const button: HTMLButtonElement = lineNumColDOM?.querySelector('button') || document.createElement('button')
        const showFromMemory = commentsVisibilityAtLineNumber?.get(Number(sourceLineNumber))
        let show = showFromMemory !== undefined ? showFromMemory : isCommentThreadResolved ? false : true

        if (!button.onclick) {
          const toggleHidden = (dom: Element) => {
            if (show) dom.setAttribute('hidden', '')
            else dom.removeAttribute('hidden')
          }
          const toggleComments = (e: KeyboardEvent | MouseEvent) => {
            let commentRow = annotatedRow.nextElementSibling as HTMLElement

            while (commentRow?.dataset?.annotatedLine) {
              // Toggle opposite place-holder as well
              const diffParent = commentRow.closest('.d2h-code-wrapper')?.parentElement
              const oppositeDiv = diffParent?.classList.contains('right')
                ? diffParent.previousElementSibling
                : diffParent?.nextElementSibling
              const oppositePlaceHolders = oppositeDiv?.querySelectorAll(
                `[data-place-holder-for-line="${sourceLineNumber}"]`
              )

              oppositePlaceHolders?.forEach(dom => toggleHidden(dom))

              toggleHidden(commentRow)
              commentRow = commentRow.nextElementSibling as HTMLElement
            }
            show = !show

            if (memorizedState) {
              commentsVisibilityAtLineNumber?.set(Number(sourceLineNumber), show)
            }

            if (!show) button.dataset.threadsCount = String(activeThreads + resolvedThreads)
            else delete button.dataset.threadsCount

            e.stopPropagation()
          }

          button.classList.add(css.toggleComment)
          button.title = getString('pr.toggleComments')
          button.dataset.toggleComment = 'true'

          document.addEventListener(PR_COMMENT_STATUS_CHANGED_EVENT, e => handleCommentStatusChange(e, toggleComments))
          button.addEventListener('keydown', e => {
            if (e.key === 'Enter') toggleComments(e)
          })
          button.onclick = toggleComments

          lineNumColDOM.appendChild(button)
        }

        let commentRow = annotatedRow.nextElementSibling as HTMLElement
        let resolvedThreads = 0
        let activeThreads = 0

        while (commentRow?.dataset?.annotatedLine) {
          if (commentRow.dataset.commentThreadStatus == CodeCommentState.RESOLVED) {
            resolvedThreads++
            if (!internalFlags.current.initialized && !showFromMemory) {
              show = false
            }
          } else activeThreads++

          commentRow = commentRow.nextElementSibling as HTMLElement
        }

        button.style.backgroundImage = `url("${activeThreads ? commentActiveIconUrl : commentResolvedIconUrl}")`

        if (!internalFlags.current.initialized) {
          internalFlags.current.initialized = true

          if (!show && resolvedThreads) button.dataset.threadsCount = String(resolvedThreads)
          else delete button.dataset.threadsCount
        }
      }

      // Cleanup the event listener on component unmount
      return () => {
        document.removeEventListener(PR_COMMENT_STATUS_CHANGED_EVENT, handleCommentStatusChange)
      }
    },
    [isCommentThreadResolved, getString, commentsVisibilityAtLineNumber, memorizedState]
  )

  const viewRefs = useRef(
    Object.fromEntries(
      commentItems.map(commentItem => [commentItem.id, createRef() as React.MutableRefObject<EditorView | undefined>])
    )
  )
  const contentRestoredRefs = useRef<Record<number, boolean>>({})

  return (
    <Render when={commentItems.length}>
      <Container className={css.viewer} padding="xlarge" ref={domRef}>
        {commentItems.map((commentItem, index) => {
          const isLastItem = index === commentItems.length - 1
          const contentFromMemorizedState = memorizedState?.uncommittedEditComments?.get(commentItem.id)
          const viewRef = viewRefs.current[commentItem.id]

          if (viewRef && contentFromMemorizedState !== undefined && !contentRestoredRefs.current[commentItem.id]) {
            editIndexes[index] = true
            contentRestoredRefs.current[commentItem.id] = true

            setTimeout(() => {
              if (contentFromMemorizedState !== commentItem.content) {
                viewRef.current?.dispatch({
                  changes: {
                    from: 0,
                    to: viewRef.current.state.doc.length,
                    insert: contentFromMemorizedState
                  }
                })
              }
            }, 0)
          }

          return (
            <ThreadSection
              key={index}
              title={
                <Layout.Horizontal
                  spacing="small"
                  style={{ alignItems: 'center' }}
                  data-outdated={commentItem?.outdated}>
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
                            className: cx(css.optionMenuIcon, css.edit),
                            iconName: 'Edit',
                            text: getString('edit'),
                            onClick: () => {
                              setEditIndexes({ ...editIndexes, ...{ [index]: true } })
                              if (memorizedState) {
                                memorizedState.uncommittedEditComments =
                                  memorizedState.uncommittedEditComments || new Map()
                                memorizedState.uncommittedEditComments.set(commentItem.id, commentItem.content)
                              }
                            }
                          },
                          {
                            hasIcon: true,
                            className: css.optionMenuIcon,
                            iconName: 'code-quote',
                            text: getString('quote'),
                            onClick: () => onQuote(commentItem?.content)
                          },
                          {
                            hasIcon: true,
                            className: css.optionMenuIcon,
                            iconName: 'code-copy',
                            text: getString('pr.copyLinkToComment'),
                            onClick: () => copyLinkToComment(commentItem.id, commentItems[0])
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
                                resetStateAtIndex(index, commentItem)
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
                  <Container className={css.outletTopOfFirstOfComment}>
                    {outlets[CommentBoxOutletPosition.TOP_OF_FIRST_COMMENT]}
                  </Container>
                </Render>

                <Match expr={editIndexes[index]}>
                  <Truthy>
                    <Container className={css.editCommentContainer} data-comment-editor-shown="true">
                      <MarkdownEditorWithPreview
                        routingId={routingId}
                        standalone={standalone}
                        repoMetadata={repoMetadata}
                        value={commentItem?.content}
                        viewRef={viewRefs.current[commentItem.id]}
                        setMentionsMap={setMentionsMap}
                        mentionsMap={mentionsMap}
                        onSave={async value => {
                          if (
                            await handleAction(
                              CommentAction.UPDATE,
                              mentionsMap ? replaceMentionEmailWithId(value, mentionsMap) : value,
                              commentItem
                            )
                          ) {
                            commentItem.content = value
                            resetStateAtIndex(index, commentItem)
                          }
                        }}
                        onChange={value => {
                          if (memorizedState) {
                            memorizedState.uncommittedEditComments = memorizedState.uncommittedEditComments || new Map()
                            memorizedState.uncommittedEditComments.set(commentItem.id, value)
                          }
                        }}
                        onCancel={() => resetStateAtIndex(index, commentItem)}
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
                        autoFocusAndPosition={contentFromMemorizedState ? false : true}
                        suggestionBlock={suggestionBlock}
                      />
                    </Container>
                  </Truthy>
                  <Else>
                    <Match expr={commentItem?.deleted}>
                      <Truthy>
                        <Text className={css.deleted}>{getString('commentDeleted')}</Text>
                      </Truthy>
                      <Else>
                        <MarkdownViewer
                          source={commentItem?.content}
                          mentions={(commentItem as CommentItem<TypesPullReqActivity>)?.payload?.mentions}
                          suggestionBlock={Object.assign(
                            {
                              commentId: commentItem.id,
                              appliedCheckSum: get(commentItem, 'payload.metadata.suggestions.applied_check_sum', ''),
                              appliedCommitSha: get(commentItem, 'payload.metadata.suggestions.applied_commit_sha', '')
                            },
                            suggestionBlock
                          )}
                          suggestionCheckSums={get(commentItem, 'payload.metadata.suggestions.check_sums', [])}
                        />
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
