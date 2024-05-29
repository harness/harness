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
import { useParams } from 'react-router-dom'
import { parseLogString } from 'pages/PullRequest/Checks/ChecksUtils'
import { lineElement, type LogLine } from 'components/LogViewer/LogViewer'
import { useGetGitspaceInstanceLogs } from 'services/cde'
import { useGetCDEAPIParams, type CDEPathParams } from 'cde/hooks/useGetCDEAPIParams'
import { getErrorMessage } from 'utils/Utils'
import { useStrings } from 'framework/strings'
import css from './GitspaceLogs.module.scss'

export const GitspaceLogs = () => {
  const { getString } = useStrings()
  const localRef = useRef<HTMLDivElement | null>()

  const { accountIdentifier, orgIdentifier, projectIdentifier } = useGetCDEAPIParams() as CDEPathParams
  const { gitspaceId } = useParams<{
    gitspaceId: string
  }>()
  const { data, loading, error, refetch } = useGetGitspaceInstanceLogs({
    accountIdentifier,
    orgIdentifier,
    projectIdentifier,
    gitspaceIdentifier: gitspaceId
  })

  useEffect(() => {
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
    <Container width={'45%'}>
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
