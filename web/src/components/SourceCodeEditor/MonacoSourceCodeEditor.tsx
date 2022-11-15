import React, { useEffect } from 'react'
import { Container } from '@harness/uicore'
import type monacoEditor from 'monaco-editor/esm/vs/editor/editor.api'
import MonacoEditor from 'react-monaco-editor'
import { noop } from 'lodash-es'
import { SourceCodeEditorProps, PLAIN_TEXT } from 'utils/Utils'

export const MonacoEditorOptions = {
  ignoreTrimWhitespace: true,
  minimap: { enabled: false },
  codeLens: false,
  scrollBeyondLastLine: false,
  smartSelect: false,
  tabSize: 2,
  insertSpaces: true,
  overviewRulerBorder: false,
  automaticLayout: true
}

const diagnosticsOptions = {
  noSemanticValidation: true,
  noSyntaxValidation: true
}

const compilerOptions = {
  jsx: 'react',
  noLib: true,
  allowNonTsExtensions: true
}

function autoAdjustEditorHeight(editor: monacoEditor.editor.IStandaloneCodeEditor) {
  // Adjust editor height based on its content
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

const toOnOff = (flag: boolean) => (flag ? 'on' : 'off')

export default function MonacoSourceCodeEditor({
  source,
  language = PLAIN_TEXT,
  lineNumbers = true,
  readOnly = false,
  className,
  height,
  autoHeight,
  wordWrap = true,
  onChange = noop
}: SourceCodeEditorProps) {
  const scrollbar = autoHeight ? 'hidden' : 'auto'

  useEffect(() => {
    monaco.languages.typescript.typescriptDefaults.setDiagnosticsOptions?.(diagnosticsOptions)
    monaco.languages.typescript.javascriptDefaults.setDiagnosticsOptions?.(diagnosticsOptions)
    monaco.languages.typescript.typescriptDefaults.setCompilerOptions(compilerOptions)
  }, [])

  return (
    <Container className={className}>
      <MonacoEditor
        language={language}
        theme="vs-light"
        value={source}
        height={height}
        options={{
          ...MonacoEditorOptions,
          readOnly,
          wordWrap: toOnOff(wordWrap),
          lineNumbers: toOnOff(lineNumbers),
          scrollbar: {
            vertical: scrollbar,
            horizontal: scrollbar,
            alwaysConsumeMouseWheel: false
          }
        }}
        editorDidMount={_editor => {
          if (autoHeight) {
            autoAdjustEditorHeight(_editor)
          }
        }}
        onChange={onChange}
      />
    </Container>
  )
}
