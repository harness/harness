import React, { Suspense } from 'react'
import { Container, Text } from '@harness/uicore'
import MarkdownEditor from '@uiw/react-markdown-editor'
import { useStrings } from 'framework/strings'
import { SourceCodeEditor } from 'components/SourceCodeEditor/SourceCodeEditor'
import './SourceCodeViewer.scss'
import type { SourceCodeEditorProps } from 'utils/Utils'

interface MarkdownViewerProps {
  source: string
}

export function MarkdownViewer({ source }: MarkdownViewerProps) {
  const { getString } = useStrings()

  return (
    <Container className="sourceCodeViewer">
      <Suspense fallback={<Text>{getString('loading')}</Text>}>
        <MarkdownEditor.Markdown
          source={source}
          skipHtml={true}
          warpperElement={{ 'data-color-mode': 'light' }}
          rehypeRewrite={(node, _index, parent) => {
            if (
              (node as unknown as HTMLDivElement).tagName === 'a' &&
              parent &&
              /^h(1|2|3|4|5|6)/.test((parent as unknown as HTMLDivElement).tagName)
            ) {
              parent.children = parent.children.slice(1)
            }
          }}
        />
      </Suspense>
    </Container>
  )
}

type SourceCodeViewerProps = Omit<SourceCodeEditorProps, 'readOnly' | 'autoHeight'>

export function SourceCodeViewer(props: SourceCodeViewerProps) {
  return <SourceCodeEditor {...props} readOnly autoHeight />
}
