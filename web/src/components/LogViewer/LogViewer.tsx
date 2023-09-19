import React, { useEffect, useRef } from 'react'
import Anser from 'anser'
import cx from 'classnames'
import { Container } from '@harnessio/uicore'
import css from './LogViewer.module.scss'

export interface LogViewerProps {
  search?: string
  content?: string
  className?: string
}

const LogTerminal: React.FC<LogViewerProps> = ({ content, className }) => {
  const ref = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    content?.split(/\r?\n/).forEach(line => ref.current?.appendChild(lineElement(line)))
  }, [content])

  return <Container ref={ref} className={cx(css.main, className)} />
}

const lineElement = (line = '') => {
  const element = document.createElement('pre')
  element.className = css.line
  element.innerHTML = Anser.ansiToHtml(line.replace(/\r?\n$/, ''))
  return element
}

export const LogViewer = React.memo(LogTerminal)
