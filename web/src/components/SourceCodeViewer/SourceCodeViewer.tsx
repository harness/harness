import React, { lazy, Suspense } from 'react'
import { Container, Text } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import { SourceCodeEditor } from 'components/SourceCodeEditor/SourceCodeEditor'
import './SourceCodeViewer.scss'
import type { SourceCodeEditorProps } from 'utils/Utils'

interface MarkdownViewerProps {
  source: string
}

export function MarkdownViewer({ source }: MarkdownViewerProps) {
  const { getString } = useStrings()
  const ReactMarkdownPreview = lazy(() => import('@uiw/react-markdown-preview'))

  return (
    <Container className="sourceCodeViewer">
      <Suspense fallback={<Text>{getString('loading')}</Text>}>
        <ReactMarkdownPreview source={source} skipHtml={true} warpperElement={{ 'data-color-mode': 'light' }} />
      </Suspense>
    </Container>
  )
}

type SourceCodeViewerProps = Omit<SourceCodeEditorProps, 'readOnly' | 'autoHeight'>

export function SourceCodeViewer(props: SourceCodeViewerProps) {
  return <SourceCodeEditor {...props} readOnly autoHeight />
}
