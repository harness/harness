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

import { ExecutionState } from 'components/ExecutionStatus/ExecutionStatus'
import type { PRChecksDecisionResult } from 'hooks/usePRChecksDecision'
import type { EnumCheckStatus } from 'services/code'
import type { GitInfoProps } from 'utils/GitUtils'

export interface ChecksProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullReqMetadata'> {
  prChecksDecisionResult?: PRChecksDecisionResult
}

type CheckType = { status: EnumCheckStatus }[]

export function findDefaultExecution<T>(collection: Iterable<T> | null | undefined) {
  return (collection as CheckType)?.length
    ? (((collection as CheckType).find(({ status }) => status === ExecutionState.ERROR) ||
        (collection as CheckType).find(({ status }) => status === ExecutionState.FAILURE) ||
        (collection as CheckType).find(({ status }) => status === ExecutionState.RUNNING) ||
        (collection as CheckType).find(({ status }) => status === ExecutionState.SUCCESS) ||
        (collection as CheckType).find(({ status }) => status === ExecutionState.PENDING) ||
        (collection as CheckType)[0]) as T)
    : null
}
export interface DetailDict {
  [key: string]: string
}

export function extractBetweenPipelinesAndExecutions(url: string) {
  const pipelinesIndex = url.indexOf('/pipelines/')
  const executionsIndex = url.indexOf('/executions/')

  if (pipelinesIndex === -1 || executionsIndex === -1) {
    return '' // Not found
  }

  const startIndex = pipelinesIndex + '/pipelines/'.length
  const endIndex = executionsIndex

  return url.substring(startIndex, endIndex)
}

export function parseLogString(logString: string) {
  if (!logString) {
    return ''
  }
  const logEntries = logString.trim().split('\n')
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const parsedLogs: any = []
  logEntries.forEach(entry => {
    // Parse the entry as JSON
    const jsonEntry = JSON.parse(entry)
    // Apply the regex to the 'out' field
    const parts = jsonEntry.out.match(/time="([^"]+)" level=([^ ]+) msg="([^"]+)"(.*)/)
    if (parts) {
      const [, time, level, message, details, out] = parts
      const detailParts = details.trim().split(' ')
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const detailDict: any = {}
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      detailParts.forEach((part: any) => {
        if (part.includes('=')) {
          const [key, value] = part.split('=')
          detailDict[key.trim()] = value.trim()
        }
      })
      parsedLogs.push({ time, level, message, out, details: detailDict, pos: jsonEntry.pos, logLevel: jsonEntry.level })
    } else {
      parsedLogs.push({
        time: jsonEntry.time,
        level: jsonEntry.level,
        message: jsonEntry.out,
        pos: jsonEntry.pos,
        logLevel: jsonEntry.level
      })
    }
  })

  return parsedLogs
}

export function capitalizeFirstLetter(str: string) {
  return str.charAt(0).toUpperCase() + str.slice(1).toLowerCase()
}
