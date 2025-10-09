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

import React, { RefObject, useCallback, useEffect, useMemo, useRef } from 'react'
import cx from 'classnames'
import { Container, useIsMounted } from '@harnessio/uicore'
import { useInView } from 'react-intersection-observer'
import type { FCWithChildren } from 'utils/types'
import { useResizeObserver } from 'hooks/useResizeObserver'
import css from './InViewDiffBlockRenderer.module.scss'

interface InViewDiffBlockRendererProps {
  root: RefObject<Element>
  blockName: string
  shouldRetainChildren: (containerDOM: HTMLElement | null) => boolean
  detectionMargin: number
}

const InViewDiffBlockRendererInternal: FCWithChildren<InViewDiffBlockRendererProps> = ({
  root,
  blockName,
  children,
  shouldRetainChildren,
  detectionMargin
}) => {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const isMounted = useIsMounted()
  const { ref, inView } = useInView({
    root: root.current,
    rootMargin: `${detectionMargin}px 0px ${detectionMargin}px 0px`,
    initialInView: false
  })
  const setContainerRef = useCallback(
    node => {
      containerRef.current = node
      ref(node)
    },
    [ref]
  )

  useResizeObserver(
    containerRef,
    useCallback(
      dom => {
        if (isMounted.current && dom) {
          dom.style.setProperty(BLOCK_HEIGHT, dom.clientHeight + 'px')
        }
      },
      [isMounted]
    )
  )

  const showChildren = useMemo(
    () => inView || shouldRetainChildren(containerRef.current),
    [inView, shouldRetainChildren]
  )

  //
  // Use `useMemo` to check if an `inView` event happens, and children is not yet rendered, then assign minHeight
  // of the container to `--block-height` value. This makes sure the scrollbar position is not jumping as when
  // children is empty, `auto` height from container reduces its height.
  // When children is rendered, we reset `minHeight` to `auto` again.
  // Important note: This can't be accomplished with `useEffect`.
  //
  useMemo(() => {
    const dom = containerRef.current

    if (dom) {
      dom.style.minHeight =
        showChildren && dom.dataset.rendered === 'false' ? dom.style.getPropertyValue(BLOCK_HEIGHT) : AUTO
    }
  }, [showChildren])

  //
  // min-height as pixel values is good when re-rendering children (null -> children) as it
  // keeps the layout and scroll position fixed (no jumping/scrollbar flickering). It becomes
  // bad when user starts interacting with components inside the diff (collapse diff, add/update
  // comments, etc...). Hence, we need to reset it back to 'auto' after children rendering is done.
  //
  useEffect(
    function resetMinHeight() {
      const dom = containerRef.current
      let timeoutId = 0

      if (dom && showChildren && dom.dataset.rendered === 'true' && dom.style.minHeight !== AUTO) {
        timeoutId = window.setTimeout(() => {
          if (isMounted.current && dom) dom.style.minHeight = AUTO
        }, RESET_MIN_HEIGHT_TIMEOUT)
      }

      return () => clearTimeout(timeoutId)
    },
    [showChildren, isMounted]
  )

  return (
    <Container
      data-block={blockName}
      data-rendered={showChildren}
      ref={setContainerRef}
      className={cx(css.main, { [css.hidden]: !inView })}>
      {showChildren ? children : null}
    </Container>
  )
}

const BLOCK_HEIGHT = '--block-height'
const AUTO = 'auto'
const RESET_MIN_HEIGHT_TIMEOUT = 2000

export const InViewDiffBlockRenderer = React.memo(InViewDiffBlockRendererInternal)
