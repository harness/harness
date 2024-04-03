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

import React, { useCallback, useEffect, useRef } from 'react'
import { PAGE_CONTAINER_WIDTH } from 'utils/Utils'
import { useEventListener } from 'hooks/useEventListener'
import { useIsSidebarExpanded } from 'hooks/useIsSidebarExpanded'
import { useAppContext } from 'AppContext'

/**
 * Hook to calculate page width and set it as `PAGE_CONTAINER_WIDTH` CSS variable.
 * This variable is used in child components to calculate their width or max-width.
 *
 * Why this is needed? We have a couple of components (Markdown, Diff Viewer, etc...)
 * which don't work well with fluid layout (auto width). They need a fixed value
 * of page width to calculate themselves correctly. The page width is never fixed.
 * It can be changed based on Nav state (expanded, collapsed), or window resize
 * event.
 */
export function useSetPageContainerWidthVar({ domRef }: { domRef: React.RefObject<HTMLElement> }) {
  // Ref to hold gap between page and viewport. In embedded version, this
  // ref value is the same as the global right bar (AIDA). Zero in standalone version
  const rightGap = useRef(0)
  const sidebarExpanded = useIsSidebarExpanded()
  const { standalone } = useAppContext()

  const setContainerWidthVar = useCallback(() => {
    const viewportWidth = window.innerWidth || document.documentElement.clientWidth
    const dom = domRef?.current

    if (dom) {
      const rect = dom.getBoundingClientRect()

      // Calculate the gap once
      if (!standalone && !rightGap.current) {
        rightGap.current = viewportWidth - rect.right
      }

      let pageWidth = viewportWidth - rightGap.current - rect.left

      if (!standalone) {
        pageWidth = Math.max(
          sidebarExpanded ? ContainerMinWidth.WITH_SIDE_BAR_EXPANDED : ContainerMinWidth.WITH_SIDE_BAR_COLLAPSED,
          pageWidth
        )
      }

      dom.style?.setProperty(PAGE_CONTAINER_WIDTH, `${pageWidth}px`)

      // After setProperty(), the actual page width could be different due to
      // min-width from a parent component of the page
      const _rect = dom.getBoundingClientRect()

      // Evaluate if that is the case, then adjust the pageWidth accordingly
      if (_rect.width > pageWidth) {
        dom.style?.setProperty(PAGE_CONTAINER_WIDTH, `${_rect.width}px`)
      }

      dom.style.maxWidth = 'var(--page-container-width)'
    }
  }, [domRef, sidebarExpanded, standalone])

  useEffect(setContainerWidthVar, [setContainerWidthVar, sidebarExpanded])
  useEventListener('resize', setContainerWidthVar)
}

const ContainerMinWidth = {
  WITH_SIDE_BAR_EXPANDED: 1008,
  WITH_SIDE_BAR_COLLAPSED: 1160
}
