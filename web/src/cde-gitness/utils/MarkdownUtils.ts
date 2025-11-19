import { marked } from 'marked'
import DOMPurify from 'dompurify'
import React from 'react'
import { Text } from '@harnessio/uicore'
import cx from 'classnames'

const sanitizeHtml = (html: string): string => {
  const sanitizedHtml = DOMPurify.sanitize(html, {
    ADD_ATTR: ['target']
  })
  return sanitizedHtml
}

export const getHTMLFromMarkdown = (markdown: string, options?: marked.MarkedOptions): string => {
  try {
    const html = marked.parse(markdown, options)
    return sanitizeHtml(html)
  } catch (e) {
    // ignore error
  }
  return ''
}

export type MarkdownTextProps = React.ComponentProps<typeof Text> & {
  text: string
  markdownClassName?: string
  sectionClassName?: string
}

export const MarkdownText: React.FC<MarkdownTextProps> = ({
  text,
  className,
  markdownClassName,
  sectionClassName,
  ...rest
}) => {
  const isMarkdown =
    text?.includes('##') ||
    text?.includes('```') ||
    text?.includes('**') ||
    text?.includes('__') ||
    text?.includes('![') ||
    (text?.includes('[') && text?.includes(']('))

  const formattedContent: string = isMarkdown ? getHTMLFromMarkdown(text) : text

  const textProps: React.ComponentProps<typeof Text> = {
    ...rest,
    className: cx(className, isMarkdown && markdownClassName)
  }

  if (isMarkdown) {
    return React.createElement(
      Text,
      textProps,
      React.createElement('div', { className: sectionClassName, dangerouslySetInnerHTML: { __html: formattedContent } })
    )
  }

  const children: React.ReactNode[] = []
  const lines = formattedContent.split('\n')
  lines.forEach((line, i) => {
    children.push(React.createElement(React.Fragment, { key: `ln-${i}` }, line))
    if (i < lines.length - 1) {
      children.push(React.createElement('br', { key: `br-${i}` }))
    }
  })

  return React.createElement(Text, textProps, children)
}
