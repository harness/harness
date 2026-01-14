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

import { useCallback, useEffect, useRef } from 'react'

interface UseScheduleJobProps<T> {
  handler: (items: T[]) => void
  initialData?: T[]
  isStreaming?: boolean
  maxProcessingBlockSize?: number
}

export function useScheduleJob<T>({
  handler,
  initialData = [],
  maxProcessingBlockSize = DEFAULT_PROCESSING_BLOCK_SIZE,
  isStreaming = false
}: UseScheduleJobProps<T>) {
  const progressRef = useRef(false)
  const rAFRef = useRef(0)
  const data = useRef(initialData)
  const scheduleJob = useCallback(() => {
    if (progressRef.current || !data.current.length) {
      return
    }

    cancelAnimationFrame(rAFRef.current)
    progressRef.current = true

    rAFRef.current = requestAnimationFrame(() => {
      try {
        // Process aggressively when data set is large, then slow down to one
        // item at a time when it is small to improve (scrolling) experience
        const size = isStreaming
          ? data.current.length > maxProcessingBlockSize
            ? maxProcessingBlockSize
            : 1
          : maxProcessingBlockSize

        // Cut a block to handler to process
        handler(data.current.splice(0, size))

        // When handler is done, turn off in-progress flag
        progressRef.current = false

        // Process the next block, if there's still data
        if (data.current.length) {
          scheduleJob()
        }
      } catch (error) {
        console.error('An error has occurred', error) // eslint-disable-line no-console
      }
    })
  }, [handler, maxProcessingBlockSize, isStreaming])

  useEffect(() => {
    return () => cancelAnimationFrame(rAFRef.current)
  }, [])

  return useCallback(
    function sendDataToScheduler(item: T | T[]) {
      if (Array.isArray(item)) {
        data.current.push(...item)
      } else if (item) {
        data.current.push(item)
      }

      scheduleJob()
    },
    [scheduleJob]
  )
}

const DEFAULT_PROCESSING_BLOCK_SIZE = 60
