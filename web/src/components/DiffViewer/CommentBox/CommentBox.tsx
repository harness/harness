import React, { useCallback, useState } from 'react'
import { useResizeDetector } from 'react-resize-detector'
import { Button, Container, ButtonVariation, Layout, Avatar, TextInput } from '@harness/uicore'
import MarkdownEditor from '@uiw/react-markdown-editor'
import { Tab, Tabs } from '@blueprintjs/core'
import { indentWithTab } from '@codemirror/commands'
import { keymap } from '@codemirror/view'
import 'highlight.js/styles/github.css'
import 'diff2html/bundles/css/diff2html.min.css'
import type { UseStringsReturn } from 'framework/strings'
import css from './CommentBox.module.scss'

interface CommentBoxProps {
  getString: UseStringsReturn['getString']
  onHeightChange: (height: number | 'auto') => void
  onCancel: () => void
  width: string
  contents?: string[]
}

export const CommentBox: React.FC<CommentBoxProps> = ({
  getString,
  onHeightChange,
  onCancel,
  width,
  contents: _contents = []
}) => {
  const [contents, setContents] = useState<string[]>(_contents)
  const [showReplyPlaceHolder, setShowReplyPlaceHolder] = useState(!!contents.length)
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
  // Note: Send 'auto' to avoid rendering flickering
  const onCancelBtnClick = useCallback(() => {
    if (!contents.length) {
      onCancel()
    } else {
      setShowReplyPlaceHolder(true)
      onHeightChange('auto')
    }
  }, [contents, setShowReplyPlaceHolder, onCancel, onHeightChange])
  const hidePlaceHolder = useCallback(() => {
    setShowReplyPlaceHolder(false)
    onHeightChange('auto')
  }, [setShowReplyPlaceHolder, onHeightChange])

  return (
    <Container className={css.main} padding="medium" width={width} ref={ref}>
      <Container className={css.box}>
        <Layout.Vertical className={css.boxLayout}>
          {!!contents.length && (
            <Container className={css.viewer} padding="xlarge">
              <Layout.Vertical spacing="large">
                {contents.map((content, index) => (
                  <MarkdownEditor.Markdown key={index} source={content} />
                ))}
              </Layout.Vertical>
            </Container>
          )}
          <Container className={css.editor}>
            {(showReplyPlaceHolder && (
              <Container>
                <Layout.Horizontal spacing="small" className={css.replyPlaceHolder} padding="medium">
                  <Avatar name="Tan Nhu" size="small" hoverCard={false} />
                  <TextInput placeholder={getString('replyHere')} onFocus={hidePlaceHolder} onClick={hidePlaceHolder} />
                </Layout.Horizontal>
              </Container>
            )) || (
              <Container padding="xlarge">
                <Layout.Vertical spacing="large">
                  <Tabs animate={true} id="CommentBoxTabs" onChange={() => onHeightChange('auto')} key="horizontal">
                    <Tabs.Expander />
                    <Tab
                      id="write"
                      title="Write"
                      panel={
                        <Container className={css.markdownEditor}>
                          <MarkdownEditor
                            value={markdown}
                            visible={false}
                            placeholder={getString(contents.length ? 'replyHere' : 'leaveAComment')}
                            theme="light"
                            indentWithTab={false}
                            autoFocus
                            toolbars={[
                              // 'header',
                              'bold',
                              'italic',
                              'strike',
                              'quote',
                              'olist',
                              'ulist',
                              'todo',
                              'link',
                              'image',
                              'codeBlock'
                            ]}
                            toolbarsMode={[]}
                            basicSetup={{
                              lineNumbers: false,
                              foldGutter: false,
                              highlightActiveLine: false
                            }}
                            extensions={[keymap.of([indentWithTab])]}
                            onChange={(value, _viewUpdate) => {
                              onHeightChange('auto')
                              setMarkdown(value)
                            }}
                          />
                        </Container>
                      }
                    />
                    <Tab
                      id="preview"
                      disabled={!markdown}
                      title="Preview"
                      panel={
                        <Container padding="large" className={css.preview}>
                          <MarkdownEditor.Markdown source={markdown} />
                        </Container>
                      }
                    />
                  </Tabs>
                  <Container>
                    <Layout.Horizontal spacing="small">
                      <Button
                        disabled={!(markdown || '').trim()}
                        variation={ButtonVariation.PRIMARY}
                        onClick={() => {
                          setContents([...contents, markdown])
                          setMarkdown('')
                          setShowReplyPlaceHolder(true)
                          onHeightChange('auto')
                        }}
                        text={getString('addComment')}
                      />
                      <Button
                        variation={ButtonVariation.TERTIARY}
                        onClick={onCancelBtnClick}
                        text={getString('cancel')}
                      />
                    </Layout.Horizontal>
                  </Container>
                </Layout.Vertical>
              </Container>
            )}
          </Container>
        </Layout.Vertical>
      </Container>
    </Container>
  )
}
