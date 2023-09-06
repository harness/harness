import React, { useEffect, useState } from 'react'
import * as monaco from 'monaco-editor'
import type monacoEditor from 'monaco-editor/esm/vs/editor/editor.api'
import MonacoEditor, { MonacoDiffEditor } from 'react-monaco-editor'
import { noop } from 'lodash-es'
import { SourceCodeEditorProps, PLAIN_TEXT } from 'utils/Utils'
import { useEventListener } from 'hooks/useEventListener'

export const MonacoEditorOptions = {
  ignoreTrimWhitespace: true,
  minimap: { enabled: false },
  codeLens: false,
  scrollBeyondLastLine: false,
  // smartSelect: false,
  tabSize: 2,
  insertSpaces: true,
  overviewRulerBorder: false,
  automaticLayout: true
}

const diagnosticsOptions = {
  noSemanticValidation: true,
  noSyntaxValidation: true
}

const compilerOptions: monacoEditor.languages.typescript.CompilerOptions = {
  jsx: monaco.languages.typescript.JsxEmit.ReactJSX,
  noLib: true,
  allowNonTsExtensions: true
}

const toOnOff = (flag: boolean) => (flag ? 'on' : 'off')

export default function MonacoSourceCodeEditor({
  source,
  language = PLAIN_TEXT,
  lineNumbers = true,
  readOnly = false,
  height,
  autoHeight,
  wordWrap = true,
  onChange = noop
}: SourceCodeEditorProps) {
  const [editor, setEditor] = useState<monacoEditor.editor.IStandaloneCodeEditor>()
  const scrollbar = autoHeight ? 'hidden' : 'auto'

  useEffect(() => {
    monaco.languages.typescript?.typescriptDefaults?.setDiagnosticsOptions?.(diagnosticsOptions)
    monaco.languages.typescript?.javascriptDefaults?.setDiagnosticsOptions?.(diagnosticsOptions)
    monaco.languages.typescript?.typescriptDefaults?.setCompilerOptions?.(compilerOptions)
  }, [])

  useEventListener('resize', () => {
    editor?.layout({ width: 0, height: 0 })
    window.requestAnimationFrame(() => editor?.layout())
  })

  return (
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
          // autoAdjustEditorHeight(_editor)
        }
        setEditor(_editor)
      }}
      onChange={onChange}
    />
  )
}

interface DiffEditorProps extends Omit<SourceCodeEditorProps, 'autoHeight'> {
  original: string
}

export function DiffEditor({
  source,
  original,
  language = PLAIN_TEXT,
  lineNumbers = true,
  readOnly = false,
  height,
  wordWrap = true,
  onChange = noop
}: DiffEditorProps) {
  const [editor, setEditor] = useState<monacoEditor.editor.IStandaloneDiffEditor>()

  useEventListener('resize', () => {
    editor?.layout({ width: 0, height: 0 })
    window.requestAnimationFrame(() => editor?.layout())
  })

  return (
    <MonacoDiffEditor
      language={language}
      theme="vs-light"
      original={original}
      value={source}
      height={height}
      options={{
        ...MonacoEditorOptions,
        smartSelect: {
          selectLeadingAndTrailingWhitespace: true
        },
        readOnly,
        wordWrap: toOnOff(wordWrap),
        lineNumbers: toOnOff(lineNumbers),
        originalEditable: false,
        scrollbar: {
          vertical: 'auto',
          horizontal: 'auto',
          alwaysConsumeMouseWheel: false
        }
      }}
      editorDidMount={setEditor}
      onChange={onChange}
    />
  )
}
