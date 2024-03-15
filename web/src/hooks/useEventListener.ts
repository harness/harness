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

import { useEffect } from 'react'

export function useEventListener<K extends keyof HTMLElementEventMap>(
  type: K,
  listener: (this: HTMLElement, ev: HTMLElementEventMap[K]) => Unknown,
  element: HTMLElement = window as unknown as HTMLElement,
  options?: boolean | AddEventListenerOptions,
  parametersCheck = () => true
) {
  useEffect(() => {
    if (parametersCheck() && element) {
      element.addEventListener(type, listener, options)
      return () => {
        element.removeEventListener(type, listener)
      }
    }
  }, [element, type, listener, options, parametersCheck])
}

export function useCustomEventListener<T = any>(
  name: string,
  listener: (event: Omit<CustomEvent, 'detail'> & { detail: T }) => void,
  parametersCheck = () => true
) {
  useEventListener(
    name as keyof HTMLElementEventMap,
    listener as (event: Event) => void,
    window as unknown as HTMLElement,
    false,
    parametersCheck
  )
}

export function dispatchCustomEvent<T>(name: string, detail: T) {
  window.dispatchEvent(new CustomEvent(name, { detail }))
}
