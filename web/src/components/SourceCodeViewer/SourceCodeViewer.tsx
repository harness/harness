import React, { lazy, Suspense, useRef } from 'react'
import { Container, Text } from '@harness/uicore'
import MonacoEditor from 'react-monaco-editor'
import { useStrings } from 'framework/strings'
import { MonacoEditorOptions } from 'utils/Utils'
import './SourceCodeViewer.scss'

interface MarkdownViewerProps {
  source: string
}

export function MarkdownViewer({ source }: MarkdownViewerProps) {
  const { getString } = useStrings()
  const ReactMarkdownPreview = lazy(() => import('@uiw/react-markdown-preview'))

  return (
    <Container className="sourceCodeViewer">
      <Suspense fallback={<Text>{getString('loading')}</Text>}>
        <ReactMarkdownPreview source={source} skipHtml={false} />
      </Suspense>
    </Container>
  )
}

interface SourceCodeViewerProps {
  source: string
  language?: string
  lineNumbers?: boolean
  highlightLines?: string // i.e: {1,3-4}, not yet supported
}

function MonacoSourceCodeViewer({ source, language = 'plaintext', lineNumbers = true }: SourceCodeViewerProps) {
  const inputContainerRef = useRef<HTMLDivElement>(null)

  return (
    <Container flex ref={inputContainerRef}>
      <MonacoEditor
        language={language}
        theme="vs-light"
        value={source}
        options={{
          ...MonacoEditorOptions,
          automaticLayout: true,
          readOnly: true,
          wordWrap: 'on',
          lineNumbers: lineNumbers ? 'on' : 'off',
          scrollbar: {
            vertical: 'hidden',
            horizontal: 'hidden',
            alwaysConsumeMouseWheel: false
          }
        }}
        editorDidMount={editor => {
          // Aadjust editor height based on content
          // https://github.com/microsoft/monaco-editor/issues/794#issuecomment-427092969
          const LINE_HEIGHT = 18
          const CONTAINER_GUTTER = 10
          const editorNode = editor.getDomNode() as HTMLElement
          const codeContainer = editorNode.getElementsByClassName('view-lines')[0]
          let prevLineCount = 0
          const adjustHeight = (): void => {
            const height =
              codeContainer.childElementCount > prevLineCount
                ? (codeContainer as HTMLElement).offsetHeight // unfold
                : codeContainer.childElementCount * LINE_HEIGHT + CONTAINER_GUTTER // fold
            prevLineCount = codeContainer.childElementCount

            editorNode.style.height = Math.max(height, 100) + 'px'
            editor.layout()
          }

          setTimeout(adjustHeight, 0)
          editor.onDidChangeModelDecorations(() => setTimeout(adjustHeight, 0))
        }}
      />
    </Container>
  )
}

export const SourceCodeViewer = React.memo(MonacoSourceCodeViewer)
