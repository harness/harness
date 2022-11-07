import React, { lazy, Suspense } from 'react'
import { Container, Text } from '@harness/uicore'
import { useStrings } from 'framework/strings'
import { SourceCodeEditor, SourceCodeEditorProps } from 'components/SourceCodeEditor/SourceCodeEditor'
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

type SourceCodeViewerProps = Omit<SourceCodeEditorProps, 'readOnly' | 'autoHeight'>

export function SourceCodeViewer(props: SourceCodeViewerProps) {
  return <SourceCodeEditor {...props} readOnly autoHeight />
}
