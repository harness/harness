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

import { useEffect, useMemo, useState } from 'react'
import { stringSubstitute } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import type { GetDataError } from 'restful-react'
import { isEqual } from 'lodash-es'
import type { GitInfoProps } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import { TypesCheck, useListStatusCheckResults, UsererrorError } from 'services/code'
import { ExecutionState } from 'components/ExecutionStatus/ExecutionStatus'

export function usePRChecksDecision({
  repoMetadata,
  pullReqMetadata
}: Partial<Pick<GitInfoProps, 'repoMetadata' | 'pullReqMetadata'>>) {
  const { getString } = useStrings()
  const { data, error, refetch } = useListStatusCheckResults({
    repo_ref: `${repoMetadata?.path as string}/+`,
    commit_sha: pullReqMetadata?.source_sha as string,
    lazy: !repoMetadata?.path || !pullReqMetadata?.source_sha
  })

  const [decisions, setDecisions] = useState<PRChecksDecisionResult | undefined>()
  // TODO: This needs to be revised, as green is shown initially
  const [complete, setComplete] = useState(true)

  useMemo(() => {
    let status: ExecutionState | undefined
    const count = { ...DEFAULT_COUNTS }
    let color = Color.GREEN_500
    let background = Color.GREEN_50
    let message = ''

    const total = data?.length

    if (total) {
      for (const check of data) {
        switch (check.status) {
          case ExecutionState.ERROR:
          case ExecutionState.FAILURE:
          case ExecutionState.RUNNING:
          case ExecutionState.PENDING:
          case ExecutionState.SUCCESS:
            count[check.status]++
            // setCount({ ...count })
            break
          default:
            console.error('Unrecognized PR check status', check) // eslint-disable-line no-console
            break
        }
      }

      if (count.error) {
        status = ExecutionState.ERROR
        color = Color.RED_900
        background = Color.RED_50
        message = stringSubstitute(getString('prChecks.error'), { count: count.error, total }) as string
      } else if (count.failure) {
        status = ExecutionState.FAILURE
        color = Color.RED_900
        background = Color.RED_50
        message = stringSubstitute(getString('prChecks.failure'), { count: count.failure, total }) as string
      } else if (count.killed) {
        status = ExecutionState.KILLED
        color = Color.RED_900
        background = Color.RED_50
        message = stringSubstitute(getString('prChecks.killed'), { count: count.killed, total }) as string
      } else if (count.running) {
        status = ExecutionState.RUNNING
        color = Color.ORANGE_900
        background = Color.ORANGE_100
        message = stringSubstitute(getString('prChecks.running'), { count: count.running, total }) as string
      } else if (count.pending) {
        status = ExecutionState.PENDING
        color = Color.GREY_600
        background = Color.GREY_100
        message = stringSubstitute(getString('prChecks.pending'), { count: count.pending, total }) as string
      } else if (count.skipped) {
        status = ExecutionState.SKIPPED
        color = Color.GREY_600
        background = Color.GREY_100
        message = stringSubstitute(getString('prChecks.skipped'), { count: count.skipped, total }) as string
      } else if (count.success) {
        status = ExecutionState.SUCCESS
        color = Color.GREEN_800
        background = Color.GREEN_50
        message = stringSubstitute(getString('prChecks.success'), { count: count.success, total }) as string
      }

      setComplete(!count.pending && !count.running)

      setDecisions(_decisions => {
        return !_decisions ||
          !isEqual(status, _decisions.overallStatus) ||
          !isEqual(count, _decisions.count) ||
          !isEqual(error, _decisions.error) ||
          !isEqual(data, _decisions.data) ||
          !isEqual(color, _decisions.color) ||
          !isEqual(background, _decisions.background) ||
          !isEqual(message, _decisions.message)
          ? {
              overallStatus: status,
              count,
              error,
              data,
              color,
              background,
              message
            }
          : _decisions
      })
    } else {
      setComplete(false)
    }
  }, [data, setDecisions]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    let tornDown = false
    const pollingFn = () => {
      if (repoMetadata?.path && pullReqMetadata?.source_sha && !complete && !tornDown) {
        // TODO: fix racing condition where an ongoing refetch of the old sha overwrites the new one.
        // TEMPORARY SOLUTION: set debounce to 1 second to reduce likelyhood
        refetch({ debounce: 1 }).then(() => {
          if (!tornDown) {
            interval = window.setTimeout(pollingFn, POLLING_INTERVAL)
          }
        })
      }
    }
    let interval = window.setTimeout(pollingFn, POLLING_INTERVAL)
    return () => {
      tornDown = true
      window.clearTimeout(interval)
    }
  }, [repoMetadata?.path, pullReqMetadata?.source_sha, complete]) // eslint-disable-line react-hooks/exhaustive-deps

  return decisions
}

export interface PRChecksDecisionResult {
  overallStatus: ExecutionState | undefined
  count: typeof DEFAULT_COUNTS
  error: GetDataError<UsererrorError> | null
  data: TypesCheck[] | null
  color: string
  background: string
  message: string
}

const POLLING_INTERVAL = 10000

const DEFAULT_COUNTS = {
  error: 0,
  failure: 0,
  pending: 0,
  running: 0,
  success: 0,
  skipped: 0,
  killed: 0
}
