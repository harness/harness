import React, { useRef } from 'react'
import { Container } from '@harness/uicore'
import MonacoEditor from 'react-monaco-editor'
import { MonacoEditorOptions } from 'utils/Utils'

export interface SourceCodeEditorProps {
  source: string
  language?: string
  lineNumbers?: boolean
  readOnly?: boolean
  highlightLines?: string // i.e: {1,3-4}, TODO: not yet supported
  className?: string
  height?: number | string
  autoHeight?: boolean
}

function MonacoSourceCodeEditor({
  source,
  language = 'plaintext',
  lineNumbers = true,
  readOnly = false,
  className,
  height,
  autoHeight
}: SourceCodeEditorProps) {
  const inputContainerRef = useRef<HTMLDivElement>(null)
  const scrollbar = autoHeight ? 'hidden' : 'auto'

  // return <Container ref={inputContainerRef} className={className}>  </Container>
  return (
    <Container ref={inputContainerRef} className={className}>
      <MonacoEditor
        language={language}
        theme="vs-light"
        value={source}
        height={height}
        options={{
          ...MonacoEditorOptions,
          ...(autoHeight ? {} : { scrollBeyondLastLine: false }),
          automaticLayout: true,
          readOnly,
          wordWrap: 'on',
          lineNumbers: lineNumbers ? 'on' : 'off',
          scrollbar: {
            vertical: scrollbar,
            horizontal: scrollbar,
            alwaysConsumeMouseWheel: false
          }
        }}
        editorDidMount={editor => {
          if (autoHeight) {
            // Aadjust editor height based on content
            // https://github.com/microsoft/monaco-editor/issues/794#issuecomment-427092969
            const LINE_HEIGHT = 18
            const CONTAINER_GUTTER = 10
            const editorNode = editor.getDomNode() as HTMLElement
            const codeContainer = editorNode.getElementsByClassName('view-lines')[0]
            let prevLineCount = 0
            const adjustHeight = (): void => {
              const _height =
                codeContainer.childElementCount > prevLineCount
                  ? (codeContainer as HTMLElement).offsetHeight // unfold
                  : codeContainer.childElementCount * LINE_HEIGHT + CONTAINER_GUTTER // fold
              prevLineCount = codeContainer.childElementCount

              editorNode.style.height = Math.max(_height, 100) + 'px'
              editor.layout()
            }

            setTimeout(adjustHeight, 0)
            editor.onDidChangeModelDecorations(() => setTimeout(adjustHeight, 0))
          }
        }}
      />
    </Container>
  )
}

export const SourceCodeEditor = React.memo(MonacoSourceCodeEditor)
