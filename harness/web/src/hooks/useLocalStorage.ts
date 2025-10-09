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

import { useState } from 'react'
import type { Dispatch, SetStateAction } from 'react'

// eslint-disable-next-line @typescript-eslint/ban-types
export function isFunction(arg: unknown): arg is Function {
  return typeof arg === 'function'
}

export function useLocalStorage<T>(key: string, initalValue: T): [T, Dispatch<SetStateAction<T>>] {
  const [state, setState] = useState(() => {
    try {
      const item = window.localStorage.getItem(key)

      return item && item !== 'undefined' ? JSON.parse(item) : initalValue
    } catch (e) {
      // eslint-disable-next-line no-console
      console.log(e)
      return initalValue
    }
  })

  function setItem(value: SetStateAction<T>) {
    try {
      const valueToSet = isFunction(value) ? value(state) : value

      setState(valueToSet)
      window.localStorage.setItem(key, JSON.stringify(valueToSet))
    } catch (e) {
      // eslint-disable-next-line no-console
      console.log(e)
    }
  }

  return [state, setItem]
}
