import React, { useCallback, useRef, useState } from 'react'
import { useResizeDetector } from 'react-resize-detector'
import type { EditorView } from '@codemirror/view'
import { Render, Match, Truthy, Falsy, Else } from 'react-jsx-match'
import { Container, Layout, Avatar, TextInput, Text, Color, FontVariation, FlexExpander } from '@harness/uicore'
import cx from 'classnames'
import ReactTimeago from 'react-timeago'
import { noop } from 'lodash-es'
import type { UseStringsReturn } from 'framework/strings'
import { ThreadSection } from 'components/ThreadSection/ThreadSection'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { useAppContext } from 'AppContext'
import { OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { MarkdownEditorWithPreview } from 'components/MarkdownEditorWithPreview/MarkdownEditorWithPreview'
import { MarkdownViewer } from 'components/MarkdownViewer/MarkdownViewer'
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

// Outlets are used to insert additional components into CommentBox
export enum CommentBoxOutletPosition {
  TOP = 'top',
  BOTTOM = 'bottom',
  TOP_OF_FIRST_COMMENT = 'top_of_first_comment',
  BOTTOM_OF_COMMENT_EDITOR = 'bottom_of_comment_editor',
  LEFT_OF_OPTIONS_MENU = 'left_of_options_menu'
}

interface CommentBoxProps<T> {
  className?: string
  getString: UseStringsReturn['getString']
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
  outlets?: Partial<Record<CommentBoxOutletPosition, React.ReactNode>>
}

export const CommentBox = <T = unknown,>({
  className,
  getString,
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
  outlets = {}
}: CommentBoxProps<T>) => {
  const [comments, setComments] = useState<CommentItem<T>[]>(commentItems)
  const [showReplyPlaceHolder, setShowReplyPlaceHolder] = useState(!!comments.length)
  const [markdown, setMarkdown] = useState(initialContent)
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
  const viewRef = useRef<EditorView>()

  return (
    <Container
      className={cx(css.main, { [css.fluid]: fluid }, className)}
      padding={!fluid ? 'medium' : undefined}
      width={width}
      ref={ref}>
      <Container className={css.box}>
        {outlets[CommentBoxOutletPosition.TOP]}

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
            outlets={outlets}
          />
          <Match expr={showReplyPlaceHolder}>
            <Truthy>
              <Container>
                <Layout.Horizontal spacing="small" className={css.replyPlaceHolder} padding="medium">
                  <Avatar name={currentUserName} size="small" hoverCard={false} />
                  <TextInput placeholder={getString('replyHere')} onFocus={hidePlaceHolder} onClick={hidePlaceHolder} />
                </Layout.Horizontal>
              </Container>
            </Truthy>
            <Falsy>
              <Container className={cx(css.newCommentContainer, { [css.hasThread]: !!comments.length })}>
                <MarkdownEditorWithPreview
                  viewRef={viewRef}
                  noBorder
                  i18n={{
                    placeHolder: getString(comments.length ? 'replyHere' : 'leaveAComment'),
                    tabEdit: getString('write'),
                    tabPreview: getString('preview'),
                    save: getString('comment'),
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
                          viewRef.current?.dispatch({
                            changes: {
                              from: 0,
                              to: viewRef.current.state.doc.length,
                              insert: ''
                            }
                          })
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
            </Falsy>
          </Match>
        </Layout.Vertical>
      </Container>
    </Container>
  )
}

interface CommentsThreadProps<T>
  extends Pick<CommentBoxProps<T>, 'commentItems' | 'getString' | 'handleAction' | 'outlets'> {
  onQuote: (content: string) => void
}

const CommentsThread = <T = unknown,>({
  getString,
  onQuote,
  commentItems = [],
  handleAction,
  outlets = {}
}: CommentsThreadProps<T>) => {
  const { standalone } = useAppContext()
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
                    <ReactTimeago date={new Date(commentItem?.updated)} />
                  </Text>

                  <Render when={commentItem?.updated !== commentItem?.created || !!commentItem?.deleted}>
                    <>
                      <PipeSeparator height={8} />
                      <Text inline font={{ variation: FontVariation.SMALL }} color={Color.GREY_400}>
                        {getString(commentItem?.deleted ? 'deleted' : 'edited')}
                      </Text>
                    </>
                  </Render>

                  <FlexExpander />
                  <Layout.Horizontal>
                    <Render when={index === 0 && outlets[CommentBoxOutletPosition.LEFT_OF_OPTIONS_MENU]}>
                      <Container padding={{ right: 'medium' }}>
                        {outlets[CommentBoxOutletPosition.LEFT_OF_OPTIONS_MENU]}
                      </Container>
                    </Render>
                    <OptionsMenuButton
                      isDark={true}
                      icon="Options"
                      iconProps={{ size: 14 }}
                      style={{ padding: '5px' }}
                      disabled={!!commentItem?.deleted}
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
                  </Layout.Horizontal>
                </Layout.Horizontal>
              }
              hideGutter={isLastItem}>
              <Container
                padding={{
                  left: editIndexes[index] ? undefined : 'medium',
                  bottom: isLastItem ? undefined : 'xsmall'
                }}>
                <Render when={index === 0 && outlets[CommentBoxOutletPosition.TOP_OF_FIRST_COMMENT]}>
                  <Container className={cx(css.outletTopOfFirstOfComment, { [css.standalone]: standalone })}>
                    {outlets[CommentBoxOutletPosition.TOP_OF_FIRST_COMMENT]}
                  </Container>
                </Render>

                <Match expr={editIndexes[index]}>
                  <Truthy>
                    <Container className={css.editCommentContainer}>
                      <MarkdownEditorWithPreview
                        value={commentItem?.content}
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
                  </Truthy>
                  <Else>
                    <Match expr={commentItem?.deleted}>
                      <Truthy>
                        <Text className={css.deleted}>{getString('commentDeleted')}</Text>
                      </Truthy>
                      <Else>
                        <MarkdownViewer source={commentItem?.content} getString={getString} />
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

const CRLF = '\n'
