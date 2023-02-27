import React, { Suspense, useCallback } from 'react'
import { Container, Text } from '@harness/uicore'
import MarkdownEditor from '@uiw/react-markdown-editor'
import rehypeVideo from 'rehype-video'
import rehypeExternalLinks from 'rehype-external-links'
import { useStrings } from 'framework/strings'
import { SourceCodeEditor } from 'components/SourceCodeEditor/SourceCodeEditor'
import type { SourceCodeEditorProps } from 'utils/Utils'
import css from './SourceCodeViewer.module.scss'

interface MarkdownViewerProps {
  source: string
  navigateTo: (url: string) => void
}

export function MarkdownViewer({ source, navigateTo }: MarkdownViewerProps) {
  const { getString } = useStrings()
  const interceptClickEventOnViewerContainer = useCallback(
    event => {
      const { target } = event

      if (target?.tagName?.toLowerCase() === 'a') {
        const { href } = target

        // Intercept click event on internal links and navigate to pages to avoid full page reload
        if (href && !href.startsWith('#')) {
          try {
            const origin = new URL(href).origin

            if (origin === window.location.origin) {
              // For some reason, history.push(href) does not work in the context of @uiw/react-markdown-editor library.
              navigateTo?.(href)
              event.stopPropagation()
              event.preventDefault()
            }
          } catch (e) {
            // eslint-disable-next-line no-console
            console.error('MarkdownViewer/interceptClickEventOnViewerContainer', e)
          }
        }
      }
    },
    [navigateTo]
  )

  return (
    <Container className={css.main} onClick={interceptClickEventOnViewerContainer}>
      <Suspense fallback={<Text>{getString('loading')}</Text>}>
        <MarkdownEditor.Markdown
          source={source}
          skipHtml={true}
          warpperElement={{ 'data-color-mode': 'light' }}
          rehypeRewrite={(node, _index, parent) => {
            if ((node as unknown as HTMLDivElement).tagName === 'a') {
              if (parent && /^h(1|2|3|4|5|6)/.test((parent as unknown as HTMLDivElement).tagName)) {
                parent.children = parent.children.slice(1)
              }
            }
          }}
          rehypePlugins={[
            rehypeVideo,
            [rehypeExternalLinks, { rel: ['nofollow noreferrer noopener'], target: '_blank' }]
          ]}
        />
      </Suspense>
    </Container>
  )
}

type SourceCodeViewerProps = Omit<SourceCodeEditorProps, 'readOnly' | 'autoHeight'>

export function SourceCodeViewer(props: SourceCodeViewerProps) {
  return <SourceCodeEditor {...props} readOnly autoHeight />
}
