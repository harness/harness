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

import { FlexExpander, Layout } from '@harnessio/uicore'
import React, { FC } from 'react'
import type { LivelogLine } from 'services/code'
import css from './ConsoleLogs.module.scss'

interface ConsoleLogsProps {
  logs: LivelogLine[]
}

export const createStreamedLogLineElement = (log: LivelogLine) => {
  const lineElement = document.createElement('div')
  lineElement.className = css.logLayout

  if (typeof log.pos === 'number') {
    const lineNumberElement = document.createElement('span')
    lineNumberElement.className = css.lineNumber
    lineNumberElement.textContent = (log.pos + 1).toString()
    lineElement.appendChild(lineNumberElement)
  }

  const logTextElement = document.createElement('span')
  logTextElement.className = css.log
  logTextElement.textContent = log.out as string
  lineElement.appendChild(logTextElement)

  const flexExpanderElement = document.createElement('span')
  flexExpanderElement.className = css.flexExpand
  lineElement.appendChild(flexExpanderElement)

  const timeElement = document.createElement('span')
  timeElement.className = css.time
  timeElement.textContent = `${log.time}s`
  lineElement.appendChild(timeElement)

  return lineElement
}

const ConsoleLogs: FC<ConsoleLogsProps> = ({ logs }) => {
  return (
    <>
      {logs.map((log, index) => {
        return (
          <Layout.Horizontal key={index} spacing={'small'} className={css.logLayout}>
            {typeof log.pos === 'number' && <span className={css.lineNumber}>{log.pos + 1}</span>}
            <span className={css.log}>{log.out}</span>
            <FlexExpander />
            <span className={css.time}>{log.time}s</span>
          </Layout.Horizontal>
        )
      })}
    </>
  )
}

export default ConsoleLogs
