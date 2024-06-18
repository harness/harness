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
import React from 'react'
import { Container } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import { ExecutionState, ExecutionStatus } from 'components/ExecutionStatus/ExecutionStatus'
import Fail from '../../../../../icons/code-fail.svg?url'
import Timeout from '../../../../../icons/code-timeout.svg?url'
import Success from '../../../../../icons/code-success.svg?url'
import css from '../PullRequestOverviewPanel.module.scss'

// Define the StatusCircle component
const StatusCircle = ({
  summary
}: {
  summary: {
    message: string
    summary: {
      failedReq: number
      pendingReq: number
      runningReq: number
      successReq: number
      failed: number
      pending: number
      running: number
      succeeded: number
      total: number
    }
  }
}) => {
  const { getString } = useStrings()
  const data = summary.summary
  const status =
    data.failedReq > 0
      ? ExecutionState.FAILURE
      : data.pendingReq > 0
      ? ExecutionState.PENDING
      : data.runningReq > 0
      ? ExecutionState.RUNNING
      : data.successReq > 0
      ? ExecutionState.SUCCESS
      : data.failed > 0
      ? ExecutionState.FAILURE
      : data.pending > 0
      ? ExecutionState.PENDING
      : data.running > 0
      ? ExecutionState.RUNNING
      : data.succeeded > 0
      ? ExecutionState.SUCCESS
      : ExecutionState.SKIPPED

  return (
    <>
      {status === ExecutionState.SUCCESS ? (
        <img alt={getString('success')} width={27} height={27} src={Success} />
      ) : (status as ExecutionState) === ExecutionState.FAILURE ? (
        <img alt={getString('failed')} width={26} height={26} src={Fail} />
      ) : (status as ExecutionState) === ExecutionState.PENDING ? (
        <img alt={getString('waiting')} width={27} height={27} src={Timeout} className={css.timeoutIcon} />
      ) : (
        <Container className={css.statusCircleContainer}>
          <ExecutionStatus
            className={css.iconStatus}
            status={status as ExecutionState}
            iconOnly
            noBackground
            iconSize={status === ExecutionState.FAILURE ? 27 : 26}
            isCi
            inPr
          />
        </Container>
      )}
    </>
  )
}

export default StatusCircle
