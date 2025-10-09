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

import { useLayoutEffect, type RefObject, useCallback } from 'react'

export function useResizeObserver<T extends Element>(ref: RefObject<T>, callback: (element: T) => void) {
  const fn = useCallback(() => callback(ref.current as T), [callback, ref])

  useLayoutEffect(() => {
    let taskId = 0
    const dom = ref.current as T

    const resizeObserver = new ResizeObserver(() => {
      cancelAnimationFrame(taskId)
      taskId = requestAnimationFrame(fn)
    })

    resizeObserver.observe(dom)

    return () => {
      cancelAnimationFrame(taskId)
      resizeObserver.unobserve(dom)
    }
  }, [ref, callback, fn])
}
