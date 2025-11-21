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

import { useCallback, useEffect, useRef, useState } from 'react'
import { CustomEventName } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { useCustomEventListener } from './useEventListener'

/**
 * This hook determines if Harness Navigation Sidebar is expanded or not.
 */
export function useIsSidebarExpanded() {
  const [isSidebarExpanded, setIsSidebarExpanded] = useState(
    (document.querySelector('[data-code-repo-section]')?.clientWidth || 0) <= 64 ? false : true
  )

  useCustomEventListener(
    CustomEventName.SIDE_NAV_EXPANDED_EVENT,
    useCallback((event: CustomEvent) => {
      setIsSidebarExpanded(_ => !!event.detail)
    }, [])
  )

  return isSidebarExpanded
}

export function useCollapseHarnessNav() {
  const { standalone } = useAppContext()
  const isSidebarExpanded = useIsSidebarExpanded()
  const handled = useRef(!standalone && isSidebarExpanded)
  const internalFlags = useRef({
    initialized: false
  })

  useEffect(() => {
    if (handled.current) {
      const nav = document.getElementById('main-side-nav')
      const pullReqNavItem = nav?.querySelector('[data-code-repo-section="pull-requests"]')
      const toggleNavButton = nav?.querySelector('span[icon][class*="icon-symbol-triangle"]') as HTMLElement

      if (pullReqNavItem && toggleNavButton) {
        const isCollapsed = pullReqNavItem.clientWidth <= 64

        if (!isCollapsed) {
          setTimeout(() => {
            toggleNavButton.click()
            internalFlags.current.initialized = true
          }, 0)
        }
      }

      return () => {
        if (handled.current && toggleNavButton) {
          toggleNavButton.click()
        }
      }
    }
  }, [])

  useEffect(() => {
    if (internalFlags.current.initialized && !isSidebarExpanded) {
      internalFlags.current.initialized = false

      if (handled.current) {
        handled.current = false
      }
    }
  }, [isSidebarExpanded])
}
