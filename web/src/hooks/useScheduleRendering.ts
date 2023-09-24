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

import { useCallback, useRef } from 'react'

interface UseSchedueRenderingProps<T> {
  renderer: (items: T[]) => void
  size?: number
  initialData?: T[]
}

export function useScheduleRendering<T>({ renderer, size = 50, initialData = [] }: UseSchedueRenderingProps<T>) {
  const progressRef = useRef(false)
  const rafRef = useRef(0)
  const data = useRef<T[]>(initialData)
  const startRendering = useCallback(() => {
    if (progressRef.current || !data.current.length) {
      return
    }

    cancelAnimationFrame(rafRef.current)
    progressRef.current = true

    rafRef.current = requestAnimationFrame(() => {
      try {
        renderer(data.current.splice(0, size))
        progressRef.current = false

        if (data.current.length) {
          startRendering()
        }
      } catch (error) {
        console.error('An error has occurred', error) // eslint-disable-line no-console
      }
    })
  }, [renderer, size])

  const sendDataToRender = useCallback(
    (item: T) => {
      data.current.push(item)
      startRendering()
    },
    [startRendering]
  )

  return sendDataToRender
}
