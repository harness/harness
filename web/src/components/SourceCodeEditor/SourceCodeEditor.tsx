import React from 'react'
import type { SourceCodeEditorProps } from 'utils/Utils'
import MonacoSourceCodeEditor from './MonacoSourceCodeEditor'

function Editor(props: SourceCodeEditorProps) {
  return <MonacoSourceCodeEditor {...props} />
}

export const SourceCodeEditor = React.memo(Editor)
