import { useHistory } from 'react-router-dom'
import React, { useCallback, useState } from 'react'
import { Container } from '@harness/uicore'
import cx from 'classnames'
import MarkdownPreview from '@uiw/react-markdown-preview'
import rehypeVideo from 'rehype-video'
import rehypeExternalLinks from 'rehype-external-links'
import { INITIAL_ZOOM_LEVEL } from 'utils/Utils'
import ImageCarousel from 'components/ImageCarousel/ImageCarousel'
import css from './MarkdownViewer.module.scss'

interface MarkdownViewerProps {
  source: string
  className?: string
  maxHeight?: string | number
}

export function MarkdownViewer({ source, className, maxHeight }: MarkdownViewerProps) {
  const [isOpen, setIsOpen] = useState<boolean>(false)
  const history = useHistory()
  const [zoomLevel, setZoomLevel] = useState(INITIAL_ZOOM_LEVEL)

  const [imgEvent, setImageEvent] = useState<string[]>([])

  const interceptClickEventOnViewerContainer = useCallback(
    event => {
      const { target } = event

      const imageArray = source.split('\n').filter(string => string.includes('![image]'))
      const imageStringArray = imageArray.map(string => {
        const imageSrc = string.split('![image]')[1]
        return imageSrc.slice(1, imageSrc.length - 1)
      })
      setImageEvent(imageStringArray)
      if (target?.tagName?.toLowerCase() === 'a') {
        const { href } = target

        // Intercept click event on internal links and navigate to pages to avoid full page reload
        if (href && !href.startsWith('#')) {
          try {
            const url = new URL(href)

            if (url.origin === window.location.origin) {
              event.stopPropagation()
              event.preventDefault()
              history.push(url.pathname)
            }
          } catch (e) {
            // eslint-disable-next-line no-console
            console.error('Error: MarkdownViewer/interceptClickEventOnViewerContainer', e)
          }
        }
      } else if (event.target.nodeName?.toLowerCase() === 'img') {
        setIsOpen(true)
      }
    },
    [history, source]
  )

  return (
    <Container
      className={cx(css.main, className)}
      onClick={interceptClickEventOnViewerContainer}
      style={{ maxHeight: maxHeight }}>
      <MarkdownPreview
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
      <ImageCarousel
        isOpen={isOpen}
        setIsOpen={setIsOpen}
        setZoomLevel={setZoomLevel}
        zoomLevel={zoomLevel}
        imgEvent={imgEvent}
      />
    </Container>
  )
}
