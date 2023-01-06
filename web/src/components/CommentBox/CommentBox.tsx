import React, { useCallback, useRef, useState } from 'react'
import { useResizeDetector } from 'react-resize-detector'
import { Container, Layout, Avatar, TextInput, Text, Color, FontVariation, FlexExpander } from '@harness/uicore'
import cx from 'classnames'
import MarkdownEditor from '@uiw/react-markdown-editor'
import ReactTimeago from 'react-timeago'
import { noop } from 'lodash-es'
import type { UseStringsReturn } from 'framework/strings'
import { ThreadSection } from 'components/ThreadSection/ThreadSection'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { MenuDivider, OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import {
  MarkdownEditorWithPreview,
  MarkdownEditorWithPreviewResetProps
} from 'components/MarkdownEditorWithPreview/MarkdownEditorWithPreview'
import css from './CommentBox.module.scss'

export interface CommentItem<T = unknown> {
  author: string
  created: string | number
  updated: string | number
  deleted: string | number
  content: string
  payload?: T // optional payload for callers to handle on callback calls
}

export enum CommentAction {
  NEW = 'new',
  UPDATE = 'update',
  REPLY = 'reply',
  DELETE = 'delete'
}

interface CommentBoxProps<T> {
  getString: UseStringsReturn['getString']
  onHeightChange?: (height: number) => void
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
}

export const CommentBox = <T = unknown,>({
  getString,
  onHeightChange = noop,
  width,
  fluid,
  commentItems = [],
  currentUserName,
  // executeDeleteComent = noop,
  handleAction,
  onCancel = noop,
  hideCancel,
  resetOnSave
}: CommentBoxProps<T>) => {
  const [comments, setComments] = useState<CommentItem<T>[]>(commentItems)
  const [showReplyPlaceHolder, setShowReplyPlaceHolder] = useState(!!comments.length)
  const [markdown, setMarkdown] = useState('')
  const { ref } = useResizeDetector<HTMLDivElement>({
    refreshMode: 'debounce',
    handleWidth: false,
    refreshRate: 50,
    observerOptions: { box: 'border-box' },
    onResize: () => onHeightChange(ref.current?.offsetHeight as number)
  })
  const _onCancel = useCallback(() => {
    setMarkdown('')
    if (!comments.length) {
      onCancel()
    } else {
      setShowReplyPlaceHolder(true)
    }
  }, [comments, setShowReplyPlaceHolder, onCancel])
  const hidePlaceHolder = useCallback(() => setShowReplyPlaceHolder(false), [setShowReplyPlaceHolder])
  const onQuote = useCallback((content: string) => {
    setShowReplyPlaceHolder(false)
    setMarkdown(
      content
        .split(CRLF)
        .map(line => `> ${line}`)
        .concat([CRLF, CRLF])
        .join(CRLF)
    )
  }, [])
  const editorRef = useRef<MarkdownEditorWithPreviewResetProps>()

  return (
    <Container
      className={cx(css.main, fluid ? css.fluid : '')}
      padding={!fluid ? 'medium' : undefined}
      width={width}
      ref={ref}>
      <Container className={css.box}>
        <Layout.Vertical>
          <CommentsThread<T>
            commentItems={comments}
            getString={getString}
            onQuote={onQuote}
            handleAction={async (action, content, atCommentItem) => {
              const [result, updatedItem] = await handleAction(action, content, atCommentItem)

              if (result && action === CommentAction.DELETE && atCommentItem) {
                atCommentItem.updated = atCommentItem.deleted = Date.now()
                setComments([...comments])
              }

              return [result, updatedItem]
            }}
          />

          {(showReplyPlaceHolder && (
            <Container>
              <Layout.Horizontal spacing="small" className={css.replyPlaceHolder} padding="medium">
                <Avatar name={currentUserName} size="small" hoverCard={false} />
                <TextInput placeholder={getString('replyHere')} onFocus={hidePlaceHolder} onClick={hidePlaceHolder} />
              </Layout.Horizontal>
            </Container>
          )) || (
            <Container padding="xlarge" className={cx(css.newCommentContainer, { [css.hasThread]: !!comments.length })}>
              <MarkdownEditorWithPreview
                editorRef={editorRef as React.MutableRefObject<MarkdownEditorWithPreviewResetProps>}
                i18n={{
                  placeHolder: getString(comments.length ? 'replyHere' : 'leaveAComment'),
                  tabEdit: getString('write'),
                  tabPreview: getString('preview'),
                  save: getString('addComment'),
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

                      if (resetOnSave) {
                        editorRef.current?.resetEditor?.()
                      } else {
                        setComments([...comments, updatedItem as CommentItem<T>])
                        setShowReplyPlaceHolder(true)
                      }
                    }
                  } else {
                    alert('handleAction must be implemented...')
                  }
                }}
                onCancel={_onCancel}
                hideCancel={hideCancel}
              />
            </Container>
          )}
        </Layout.Vertical>
      </Container>
    </Container>
  )
}

interface CommentsThreadProps<T> extends Pick<CommentBoxProps<T>, 'commentItems' | 'getString' | 'handleAction'> {
  onQuote: (content: string) => void
}

const CommentsThread = <T = unknown,>({
  getString,
  onQuote,
  commentItems = [],
  handleAction
}: CommentsThreadProps<T>) => {
  const [editIndexes, setEditIndexes] = useState<Record<number, boolean>>({})
  const resetStateAtIndex = useCallback(
    (index: number) => {
      delete editIndexes[index]
      setEditIndexes({ ...editIndexes })
    },
    [editIndexes]
  )

  return commentItems.length ? (
    <Container className={css.viewer} padding="xlarge">
      {commentItems.map((commentItem, index) => {
        const isLastItem = index === commentItems.length - 1

        return (
          <ThreadSection
            key={index}
            title={
              <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }}>
                <Text inline icon="code-chat"></Text>
                <Avatar name={commentItem.author} size="small" hoverCard={false} />
                <Text inline>
                  <strong>{commentItem.author}</strong>
                </Text>
                <PipeSeparator height={8} />
                <Text inline font={{ variation: FontVariation.SMALL }} color={Color.GREY_400}>
                  <ReactTimeago date={new Date(commentItem.updated)} />
                </Text>
                {(commentItem.updated !== commentItem.created || !!commentItem.deleted) && (
                  <>
                    <PipeSeparator height={8} />
                    <Text inline font={{ variation: FontVariation.SMALL }} color={Color.GREY_400}>
                      {getString(commentItem.deleted ? 'deleted' : 'edited')}
                    </Text>
                  </>
                )}
                <FlexExpander />
                <OptionsMenuButton
                  isDark={false}
                  icon="Options"
                  iconProps={{ size: 14 }}
                  style={{ padding: '5px' }}
                  disabled={!!commentItem.deleted}
                  items={[
                    {
                      text: getString('edit'),
                      onClick: () => setEditIndexes({ ...editIndexes, ...{ [index]: true } })
                    },
                    {
                      text: getString('quote'),
                      onClick: () => onQuote(commentItem.content)
                    },
                    MenuDivider,
                    {
                      text: (
                        <Text width={100} color={Color.RED_500}>
                          {getString('delete')}
                        </Text>
                      ),
                      onClick: async () => {
                        if (await handleAction(CommentAction.DELETE, '', commentItem)) {
                          resetStateAtIndex(index)
                        }
                      },
                      className: css.deleteMenuItem
                    }
                  ]}
                />
              </Layout.Horizontal>
            }
            hideGutter={isLastItem}>
            <Container
              padding={{ left: editIndexes[index] ? undefined : 'medium', bottom: isLastItem ? undefined : 'xsmall' }}>
              {editIndexes[index] ? (
                <Container className={css.editCommentContainer}>
                  <MarkdownEditorWithPreview
                    value={commentItem.content}
                    onSave={async value => {
                      if (await handleAction(CommentAction.UPDATE, value, commentItem)) {
                        commentItem.content = value
                        resetStateAtIndex(index)
                      }
                    }}
                    onCancel={() => resetStateAtIndex(index)}
                    i18n={{
                      placeHolder: getString('leaveAComment'),
                      tabEdit: getString('write'),
                      tabPreview: getString('preview'),
                      save: getString('save'),
                      cancel: getString('cancel')
                    }}
                  />
                </Container>
              ) : (
                (!commentItem.deleted && <MarkdownEditor.Markdown source={commentItem.content} />) || (
                  <Text className={css.deleted}>{getString('commentDeleted')}</Text>
                )
              )}
            </Container>
          </ThreadSection>
        )
      })}
    </Container>
  ) : null
}

const CRLF = '\n'
