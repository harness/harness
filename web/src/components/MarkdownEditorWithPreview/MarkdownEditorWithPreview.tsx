import React, { useState } from 'react'
import { Button, Container, ButtonVariation, Layout } from '@harness/uicore'
import MarkdownEditor from '@uiw/react-markdown-editor'
import { Tab, Tabs } from '@blueprintjs/core'
import { indentWithTab } from '@codemirror/commands'
import cx from 'classnames'
import { keymap, EditorView } from '@codemirror/view'
import { noop } from 'lodash-es'
import 'highlight.js/styles/github.css'
import 'diff2html/bundles/css/diff2html.min.css'
import type { IToolBarProps } from '@uiw/react-markdown-editor/cjs/components/ToolBar'
import css from './MarkdownEditorWithPreview.module.scss'

interface MarkdownEditorWithPreviewProps {
  placeHolder: string
  value: string
  editTabText: string
  previewTabText: string
  cancelButtonText: string
  saveButtonText: string
  onChange?: (value: string, original: string) => void
  onSave: (value: string, original: string) => void
  onCancel: () => void
}

export function MarkdownEditorWithPreview({
  placeHolder,
  value,
  editTabText,
  previewTabText,
  saveButtonText,
  cancelButtonText,
  onChange = noop,
  onSave,
  onCancel
}: MarkdownEditorWithPreviewProps) {
  const [original] = useState(value)
  const [selectedTab, setSelectedTab] = useState<MarkdownEditorTab>(MarkdownEditorTab.WRITE)
  const [val, setVal] = useState(value)

  return (
    <Container className={cx(css.main, selectedTab === MarkdownEditorTab.PREVIEW ? css.withPreview : '')}>
      <Layout.Vertical spacing="large">
        <Tabs
          className={css.tabs}
          defaultSelectedTabId={selectedTab}
          onChange={tabId => setSelectedTab(tabId as MarkdownEditorTab)}>
          <Tab
            id={MarkdownEditorTab.WRITE}
            title={editTabText}
            panel={
              <Container className={css.markdownEditor}>
                <MarkdownEditor
                  value={val}
                  visible={false}
                  placeholder={placeHolder}
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
            title={previewTabText}
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
              text={saveButtonText}
            />
            <Button variation={ButtonVariation.TERTIARY} onClick={onCancel} text={cancelButtonText} />
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
