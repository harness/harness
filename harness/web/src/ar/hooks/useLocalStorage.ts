/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { Dispatch, SetStateAction, useEffect, useState } from 'react'

export function useLocalStorage<T>(
  key: string,
  initalValue: T,
  storage: Storage = window.localStorage
): [T, Dispatch<SetStateAction<T>>] {
  const [state, setState] = useState(() => {
    try {
      const item = storage.getItem(key)

      return item && item !== 'undefined' ? JSON.parse(item) : initalValue
    } catch (e) {
      // eslint-disable-next-line no-console
      console.log(e)
      return initalValue
    }
  })

  useEffect(() => {
    try {
      storage.setItem(key, JSON.stringify(state))
    } catch (e) {
      // eslint-disable-next-line no-console
      console.log(e)
    }
  }, [state])

  function setItem(value: SetStateAction<T>): void {
    setState(value)
  }

  return [state, setItem]
}
