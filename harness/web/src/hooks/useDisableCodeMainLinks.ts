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

export function useDisableCodeMainLinks(disabled: boolean) {
  useEffect(() => {
    if (disabled) {
      const nav = document.querySelectorAll('[data-code-repo-section]')

      nav?.forEach(element => {
        if (
          element.getAttribute('data-code-repo-section') !== 'files' &&
          element.getAttribute('data-code-repo-section') !== 'settings'
        ) {
          element.setAttribute('aria-disabled', 'true')
        }
      })

      return () => {
        nav?.forEach(element => {
          element.removeAttribute('aria-disabled')
        })
      }
    }
  }, [disabled])
}
