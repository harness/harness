import React from 'react'
import { SourceCodeEditor } from 'components/SourceCodeEditor/SourceCodeEditor'
import type { SourceCodeEditorProps } from 'utils/Utils'

type SourceCodeViewerProps = Omit<SourceCodeEditorProps, 'readOnly' | 'autoHeight'>

export function SourceCodeViewer(props: SourceCodeViewerProps) {
  return <SourceCodeEditor {...props} readOnly autoHeight />
}
