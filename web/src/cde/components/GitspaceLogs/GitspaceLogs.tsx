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

import React, { useRef, useEffect } from 'react'
import cx from 'classnames'
import { Text, Container, PageError } from '@harnessio/uicore'
import type { GetDataError, UseGetProps } from 'restful-react'
import { lineElement, type LogLine } from 'components/LogViewer/LogViewer'
import type { OpenapiGetGitspaceLogsResponse } from 'services/cde'
import { getErrorMessage } from 'utils/Utils'
import { useStrings } from 'framework/strings'
import { parseLogString } from './GitspaceLogs.utils'
import css from './GitspaceLogs.module.scss'

interface GitlogsProps {
  data: OpenapiGetGitspaceLogsResponse
  loading?: boolean
  error?: GetDataError<unknown> | null
  refetch: (
    options?: Partial<Omit<UseGetProps<OpenapiGetGitspaceLogsResponse, unknown, void, unknown>, 'lazy'>> | undefined
  ) => Promise<void>
}

export const GitspaceLogs = ({ data, loading, error, refetch }: GitlogsProps) => {
  const { getString } = useStrings()
  const localRef = useRef<HTMLDivElement | null>()

  useEffect(() => {
    try {
      if (data) {
        const pipelineArr = parseLogString(data)
        const fragment = new DocumentFragment()
        const logContainer = localRef.current as HTMLDivElement
        // Clear the container first
        if (localRef.current) {
          localRef.current.innerHTML = ''
        }
        if (pipelineArr) {
          pipelineArr?.forEach((line: LogLine) => {
            const linePos = line.pos + 1
            const localDate = new Date(line.time)
            // Format date to a more readable format (local time)
            const formattedDate = localDate.toLocaleString()
            fragment.appendChild(
              lineElement(`${linePos}  ${line.logLevel}  ${formattedDate.replace(',', '')}  ${line.message}`)
            )
          })

          logContainer.appendChild(fragment)
        }

        const scrollParent = logContainer.parentElement as HTMLDivElement
        const autoScroll =
          scrollParent && scrollParent.scrollTop === scrollParent.scrollHeight - scrollParent.offsetHeight

        if (autoScroll || scrollParent.scrollTop === 0) {
          scrollParent.scrollTop = scrollParent.scrollHeight
        }
      }
    } catch (_err) {
      // console.log('err', err)
    }
  }, [data])

  let logMessage = ''

  if (loading) {
    logMessage = getString('cde.details.fetchingLogs')
  } else if (error) {
    logMessage = getErrorMessage(error) || getString('cde.details.logsFailed')
  } else {
    logMessage = getString('cde.details.noLogsFound')
  }

  return (
    <Container className={css.logContainer}>
      <Text className={css.logTitle}>{getString('cde.logs')}</Text>
      <Container className={css.consoleContainer}>
        {data ? (
          <Container key={`harnesslog`} ref={localRef} className={cx(css.mainLog, css.stepLogContainer)} />
        ) : (
          <Container key={`harnesslog`} ref={localRef} className={cx(css.mainLog, css.stepLogContainer)}>
            {error ? (
              <PageError onClick={() => refetch()} message={getErrorMessage(error)} />
            ) : (
              <Text>{logMessage}</Text>
            )}
          </Container>
        )}
      </Container>
    </Container>
  )
}
