/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { useHistory } from 'react-router-dom'
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Container } from '@harnessio/uicore'
import { isEmpty } from 'lodash-es'
import cx from 'classnames'
import MarkdownPreview from '@uiw/react-markdown-preview'
import rehypeVideo from 'rehype-video'
import rehypeExternalLinks, { Element } from 'rehype-external-links'
import { INITIAL_ZOOM_LEVEL, generateAlphaNumericHash } from 'utils/Utils'
import ImageCarousel from 'components/ImageCarousel/ImageCarousel'
import css from './MarkdownViewer.module.scss'

interface MarkdownViewerProps {
  source: string
  inDescriptionBox?: boolean
  className?: string
  maxHeight?: string | number
  darkMode?: boolean
  handleDescUpdate?: (payload: string) => void
  setOriginalContent?: React.Dispatch<React.SetStateAction<string>>
}

export function MarkdownViewer({
  source,
  className,
  maxHeight,
  darkMode,

  setOriginalContent,
  handleDescUpdate,
  inDescriptionBox = false
}: MarkdownViewerProps) {
  const [isOpen, setIsOpen] = useState<boolean>(false)
  const history = useHistory()
  const [zoomLevel, setZoomLevel] = useState(INITIAL_ZOOM_LEVEL)
  const [imgEvent, setImageEvent] = useState<string[]>([])
  const refRootHref = useMemo(() => document.getElementById('repository-ref-root')?.getAttribute('href'), [])
  const ref = useRef<HTMLDivElement>()
  const [markdown, setMarkdown] = useState(source)

  const interceptClickEventOnViewerContainer = useCallback(
    event => {
      const imgTags = ref?.current?.querySelector('.wmde-markdown')?.querySelectorAll('img')
      const { target } = event
      if (imgTags && !isEmpty(imgTags)) {
        const imageArray = Array.from(imgTags)
        const imageStringArray = imageArray.filter(object => object.src && !object.className).map(img => img.src)
        setImageEvent(imageStringArray)
      }

      if (target?.tagName?.toLowerCase() === 'a') {
        const href = target.getAttribute('href')

        // Intercept click event on internal links and navigate to pages to avoid full page reload
        if (href && !/^http(s)?:\/\//.test(href)) {
          try {
            const url = new URL(target.href)

            if (url.origin === window.location.origin) {
              event.stopPropagation()
              event.preventDefault()

              if (href.startsWith('#')) {
                document.getElementById(href.slice(1).toLowerCase())?.scrollIntoView()
              } else {
                history.push(url.pathname)
              }
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
    [history]
  )
  const [flag, setFlag] = useState(false)
  const handleCheckboxChange = useCallback(
    async (lineNumber: number) => {
      const newMarkdown = source
        .split('\n')
        .map((line, index) => {
          if (index === lineNumber) {
            return line.startsWith('- [ ]') ? line.replace('- [ ]', '- [x]') : line.replace('- [x]', '- [ ]')
          }
          return line
        })
        .join('\n')

      setOriginalContent?.(newMarkdown)
      setFlag(true)
      setMarkdown(newMarkdown)
      handleDescUpdate?.(newMarkdown)
    },
    [source]
  )

  useEffect(() => {
    const handleClick = (e: MouseEvent) => {
      const target = e.target as HTMLInputElement
      if (target.type === 'checkbox') {
        const lineNumber = parseInt(target.getAttribute('data-line-number') || '0', 10)
        handleCheckboxChange(lineNumber)
      }
    }

    document.addEventListener('click', handleClick)
    return () => {
      document.removeEventListener('click', handleClick)
    }
  }, [source])
  const hash = generateAlphaNumericHash(6)

  return (
    <Container
      className={cx(css.main, className, { [css.withMaxHeight]: maxHeight && maxHeight > 0 })}
      onClick={interceptClickEventOnViewerContainer}
      style={{ maxHeight: maxHeight }}
      ref={ref}>
      <MarkdownPreview
        key={flag ? hash : 0}
        source={markdown}
        skipHtml={true}
        warpperElement={{ 'data-color-mode': darkMode ? 'dark' : 'light' }}
        rehypeRewrite={(node, _index, parent) => {
          if ((node as unknown as HTMLDivElement).tagName === 'a') {
            if (parent && /^h(1|2|3|4|5|6)/.test((parent as unknown as HTMLDivElement).tagName)) {
              parent.children = parent.children.slice(1)
            }

            // Rewrite a.href to point to the correct location for relative links to files inside repository.
            // Relative links are defined as links that do not start with /, #, https:, http:, mailto:,
            // tel:, data:, javascript:, sms:, or http(s):
            if (refRootHref) {
              const { properties } = node as unknown as { properties: { href: string } }
              let href: string = properties.href

              if (
                href &&
                !href.startsWith('/') &&
                !href.startsWith('#') &&
                !href.startsWith('https:') &&
                !href.startsWith('http:') &&
                !href.startsWith('mailto:') &&
                !href.startsWith('tel:') &&
                !href.startsWith('data:') &&
                !href.startsWith('javascript:') &&
                !href.startsWith('sms:') &&
                !/^http(s)?:/.test(href)
              ) {
                try {
                  // Some relative links are prefixed by `./`, normalize them
                  if (href.startsWith('./')) {
                    href = properties.href = properties.href.replace('./', '')
                  }

                  // Test if the link is relative to the current page.
                  // If true, rewrite it to point to the correct location
                  if (new URL(window.location.href + '/' + href).origin === window.location.origin) {
                    const currentPath = window.location.href.split('~/')[1]
                    const replaceReadmeText = currentPath?.replace('README.md', '') ?? ''
                    const newUrl =
                      '/~/' + (currentPath && !currentPath.includes(href) ? replaceReadmeText + '/' : '') + href
                    properties.href = (refRootHref + newUrl.replace(/\/\//g, '/')).replace(/^\/ng\//, '/')
                  }
                } catch (_exception) {
                  // eslint-disable-line no-empty
                }
              }
            }
          }
          if (
            (node as unknown as HTMLDivElement).tagName === 'input' &&
            (node as Unknown as Element)?.properties?.type === 'checkbox'
          ) {
            const lineNumber = parent?.position?.start?.line ? parent?.position?.start?.line - 1 : 0
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const element = node as any
            element.properties['data-line-number'] = lineNumber.toString()
            element.properties.disabled = !inDescriptionBox
          }
        }}
        rehypePlugins={[
          [rehypeVideo, { test: /\/(.*)(.mp4|.mov|.webm|.mkv|.flv)$/, details: null }],
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
