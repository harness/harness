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
