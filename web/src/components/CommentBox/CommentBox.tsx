import React, { useCallback, useState } from 'react'
import { useResizeDetector } from 'react-resize-detector'
import { Container, Layout, Avatar, TextInput, Text, Color, FontVariation, FlexExpander } from '@harness/uicore'
import MarkdownEditor from '@uiw/react-markdown-editor'
import ReactTimeago from 'react-timeago'
import 'highlight.js/styles/github.css'
import 'diff2html/bundles/css/diff2html.min.css'
import type { UseStringsReturn } from 'framework/strings'
import { ThreadSection } from 'components/ThreadSection/ThreadSection'
import { PipeSeparator } from 'components/PipeSeparator/PipeSeparator'
import { CodeIcon } from 'utils/GitUtils'
import type { CommentThreadEntry, UserProfile } from 'utils/types'
import { MenuDivider, OptionsMenuButton } from 'components/OptionsMenuButton/OptionsMenuButton'
import { MarkdownEditorWithPreview } from 'components/MarkdownEditorWithPreview/MarkdownEditorWithPreview'
import css from './CommentBox.module.scss'

interface CommentBoxProps {
  getString: UseStringsReturn['getString']
  onHeightChange: (height: number) => void
  onCancel: () => void
  width?: string
  commentsThread: CommentThreadEntry[]
  currentUser: UserProfile
  executeDeleteComent: (params: { commentEntry: CommentThreadEntry; onSuccess: () => void }) => void
}

export const CommentBox: React.FC<CommentBoxProps> = ({
  getString,
  onHeightChange,
  onCancel,
  width,
  commentsThread: _commentsThread = [],
  currentUser,
  executeDeleteComent
}) => {
  // TODO: \r\n for Windows or based on configuration
  // @see https://www.aleksandrhovhannisyan.com/blog/crlf-vs-lf-normalizing-line-endings-in-git/
  const CRLF = '\n'
  const [commentsThread, setCommentsThread] = useState<CommentThreadEntry[]>(_commentsThread)
  const [showReplyPlaceHolder, setShowReplyPlaceHolder] = useState(!!commentsThread.length)
  const [markdown, setMarkdown] = useState('')
  const { ref } = useResizeDetector({
    refreshMode: 'debounce',
    handleWidth: false,
    refreshRate: 50,
    observerOptions: {
      box: 'border-box'
    },
    onResize: () => {
      onHeightChange(ref.current?.offsetHeight)
    }
  })
  const _onCancel = useCallback(() => {
    setMarkdown('')
    if (!commentsThread.length) {
      onCancel()
    } else {
      setShowReplyPlaceHolder(true)
      // onHeightChange('auto')
    }
  }, [commentsThread, setShowReplyPlaceHolder, onCancel])
  const hidePlaceHolder = useCallback(() => {
    setShowReplyPlaceHolder(false)
    // onHeightChange('auto')
  }, [setShowReplyPlaceHolder])
  const onQuote = useCallback((content: string) => {
    setShowReplyPlaceHolder(false)
    // onHeightChange('auto')
    setMarkdown(
      content
        .split(CRLF)
        .map(line => `> ${line}`)
        .concat([CRLF, CRLF])
        .join(CRLF)
    )
  }, [])

  return (
    <Container className={css.main} padding="medium" width={width} ref={ref}>
      <Container className={css.box}>
        <Layout.Vertical className={css.boxLayout}>
          <CommentsThread
            commentsThread={commentsThread}
            getString={getString}
            currentUser={currentUser}
            onQuote={onQuote}
            executeDeleteComent={executeDeleteComent}
          />

          {(showReplyPlaceHolder && (
            <Container>
              <Layout.Horizontal spacing="small" className={css.replyPlaceHolder} padding="medium">
                <Avatar name={currentUser.name} size="small" hoverCard={false} />
                <TextInput placeholder={getString('replyHere')} onFocus={hidePlaceHolder} onClick={hidePlaceHolder} />
              </Layout.Horizontal>
            </Container>
          )) || (
            <Container padding="xlarge" className={css.newCommentContainer}>
              <MarkdownEditorWithPreview
                placeHolder={getString(commentsThread.length ? 'replyHere' : 'leaveAComment')}
                value={markdown}
                editTabText={getString('write')}
                previewTabText={getString('preview')}
                saveButtonText={getString('addComment')}
                cancelButtonText={getString('cancel')}
                onChange={setMarkdown}
                onSave={value => {
                  setCommentsThread([
                    ...commentsThread,
                    {
                      id: '0',
                      author: currentUser.name,
                      created: Date.now().toString(),
                      updated: Date.now().toString(),
                      content: value
                    }
                  ])
                  setMarkdown('')
                  setShowReplyPlaceHolder(true)
                }}
                onCancel={_onCancel}
              />
            </Container>
          )}
        </Layout.Vertical>
      </Container>
    </Container>
  )
}

interface CommentsThreadProps
  extends Pick<CommentBoxProps, 'commentsThread' | 'getString' | 'currentUser' | 'executeDeleteComent'> {
  onQuote: (content: string) => void
}

const CommentsThread: React.FC<CommentsThreadProps> = ({
  getString,
  currentUser,
  onQuote,
  commentsThread = [],
  executeDeleteComent
}) => {
  const [editIndexes, setEditIndexes] = useState<Record<number, boolean>>({})

  return commentsThread.length ? (
    <Container className={css.viewer} padding="xlarge">
      {commentsThread.map((commentEntry, index) => {
        const isLastItem = index === commentsThread.length - 1

        return (
          <ThreadSection
            key={index}
            title={
              <Layout.Horizontal spacing="small" style={{ alignItems: 'center' }}>
                <Text inline icon={CodeIcon.Chat}></Text>
                <Avatar name={commentEntry.author} size="small" hoverCard={false} />
                <Text inline>
                  <strong>{commentEntry.author}</strong>
                </Text>
                <PipeSeparator height={8} />
                <Text inline font={{ variation: FontVariation.SMALL }} color={Color.GREY_400}>
                  <ReactTimeago date={new Date(commentEntry.updated)} />
                </Text>
                <FlexExpander />
                <OptionsMenuButton
                  isDark={false}
                  icon="Options"
                  iconProps={{ size: 14 }}
                  style={{ padding: '5px' }}
                  items={[
                    {
                      text: getString('edit'),
                      onClick: () => setEditIndexes({ ...editIndexes, ...{ [index]: true } })
                    },
                    {
                      text: getString('quote'),
                      onClick: () => {
                        onQuote(commentEntry.content)
                      }
                    },
                    MenuDivider,
                    {
                      text: (
                        <Text width={100} color={Color.RED_500}>
                          {getString('delete')}
                        </Text>
                      ),
                      onClick: () =>
                        executeDeleteComent({
                          commentEntry,
                          onSuccess: () => {
                            alert('success')
                          }
                        }),
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
                    placeHolder={getString('leaveAComment')}
                    value={commentEntry.content}
                    editTabText={getString('write')}
                    previewTabText={getString('preview')}
                    saveButtonText={getString('save')}
                    cancelButtonText={getString('cancel')}
                    onSave={(value, original) => {}}
                    onCancel={() => {
                      delete editIndexes[index]
                      setEditIndexes({ ...editIndexes })
                    }}
                  />
                </Container>
              ) : (
                <MarkdownEditor.Markdown key={index} source={commentEntry.content} />
              )}
            </Container>
          </ThreadSection>
        )
      })}
    </Container>
  ) : null
}
