import React, { useEffect, useState } from 'react'
import { Button, Container, ButtonVariation, Layout } from '@harness/uicore'
import MarkdownEditor from '@uiw/react-markdown-editor'
import { Tab, Tabs } from '@blueprintjs/core'
import { indentWithTab } from '@codemirror/commands'
import cx from 'classnames'
import { keymap, EditorView } from '@codemirror/view'
import { noop } from 'lodash-es'
import type { IToolBarProps } from '@uiw/react-markdown-editor/cjs/components/ToolBar'
import css from './MarkdownEditorWithPreview.module.scss'

export interface MarkdownEditorWithPreviewResetProps {
  resetEditor: () => void
}

interface MarkdownEditorWithPreviewProps {
  value: string
  onChange?: (value: string, original: string) => void
  onSave: (value: string, original: string) => void
  onCancel: () => void
  i18n: {
    placeHolder: string
    tabEdit: string
    tabPreview: string
    cancel: string
    save: string
  }
  hideCancel?: boolean
  maxEditorHeight?: string
  editorRef?: React.MutableRefObject<MarkdownEditorWithPreviewResetProps>
}

export function MarkdownEditorWithPreview({
  value,
  onChange = noop,
  onSave,
  onCancel,
  i18n,
  hideCancel,
  maxEditorHeight,
  editorRef
}: MarkdownEditorWithPreviewProps) {
  const [original, setOriginal] = useState(value)
  const [selectedTab, setSelectedTab] = useState<MarkdownEditorTab>(MarkdownEditorTab.WRITE)
  const [val, setVal] = useState(value)

  useEffect(() => {
    if (editorRef) {
      editorRef.current = {
        resetEditor: () => {
          setOriginal(value)
          setVal(value)
        }
      }
      editorRef.current.resetEditor()
    }
  }, [editorRef]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <Container className={cx(css.main, selectedTab === MarkdownEditorTab.PREVIEW ? css.withPreview : '')}>
      <Layout.Vertical spacing="large">
        <Tabs
          className={css.tabs}
          defaultSelectedTabId={selectedTab}
          onChange={tabId => setSelectedTab(tabId as MarkdownEditorTab)}>
          <Tab
            id={MarkdownEditorTab.WRITE}
            title={i18n.tabEdit}
            panel={
              <Container
                className={css.markdownEditor}
                style={{ '--max-editor-height': maxEditorHeight } as React.CSSProperties}>
                <MarkdownEditor
                  value={val}
                  visible={false}
                  placeholder={i18n.placeHolder}
                  theme="light"
                  indentWithTab={false}
                  autoFocus
                  // TODO: Customize toolbars to show tooltip.
                  // @see https://github.com/uiwjs/react-markdown-editor#custom-toolbars
                  toolbars={toolbars}
                  toolbarsMode={[]}
                  basicSetup={{
                    lineNumbers: false,
                    foldGutter: false,
                    highlightActiveLine: false
                  }}
                  extensions={[keymap.of([indentWithTab]), EditorView.lineWrapping]}
                  onChange={(_value, _viewUpdate) => {
                    setVal(_value)
                    onChange(_value, original)
                  }}
                />
              </Container>
            }
          />
          <Tab
            id={MarkdownEditorTab.PREVIEW}
            disabled={!value}
            title={i18n.tabPreview}
            panel={
              <Container padding="large" className={css.preview}>
                <MarkdownEditor.Markdown source={val} />
              </Container>
            }
          />
        </Tabs>
        <Container>
          <Layout.Horizontal spacing="small">
            <Button
              disabled={!(val || '').trim() || val === original}
              variation={ButtonVariation.PRIMARY}
              onClick={() => onSave(val, original)}
              text={i18n.save}
            />
            {!hideCancel && <Button variation={ButtonVariation.TERTIARY} onClick={onCancel} text={i18n.cancel} />}
          </Layout.Horizontal>
        </Container>
      </Layout.Vertical>
    </Container>
  )
}

const toolbars: IToolBarProps['toolbars'] = ['bold', 'strike', 'olist', 'ulist', 'todo', 'link', 'image']

enum MarkdownEditorTab {
  WRITE = 'write',
  PREVIEW = 'preview'
}
