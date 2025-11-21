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

/**
 * Based on https://github.com/contiamo/restful-react/blob/7aa3d75694f919d0317981a128b139abe163e08e/src/util/useDeepCompareEffect.ts
 */
import React, { useEffect, useMemo, useRef } from 'react'
import { isEqualWith } from 'lodash-es'

/**
 * Custom version of isEqual to handle function comparison
 */
function isEqual(a: unknown, b: unknown): boolean {
  return isEqualWith(a, b, (x: unknown, y: unknown): boolean | undefined => {
    // Deal with the function comparison case
    if (typeof x === 'function' && typeof y === 'function') {
      return x.toString() === y.toString()
    }

    // Fallback on the method
    return undefined
  })
}

function useDeepCompareMemoize(value: React.DependencyList): React.DependencyList | undefined {
  const ref = useRef<React.DependencyList>()

  if (!isEqual(value, ref.current)) {
    ref.current = value
  }

  return ref.current
}

/**
 * Accepts a function that contains imperative, possibly effectful code.
 *
 * This is the deepCompare version of the `React.useEffect` hooks (that is shallowed compare)
 *
 * @param effect Imperative function that can return a cleanup function
 * @param deps If present, effect will only activate if the values in the list change.
 *
 * @see https://gist.github.com/kentcdodds/fb8540a05c43faf636dd68647747b074#gistcomment-2830503
 */
export function useDeepCompareEffect(effect: React.EffectCallback, deps: React.DependencyList): void {
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(effect, useDeepCompareMemoize(deps))
}

/**
 * Accepts a function that contains imperative, possibly effectful code.
 *
 * This is the deepCompare version of the `React.useMemo` hooks (that is shallowed compare)
 *
 * @param effect Imperative function that can return a useMemo value
 * @param deps If present, effect will only activate if the values in the list change.
 *
 * @see https://gist.github.com/kentcdodds/fb8540a05c43faf636dd68647747b074#gistcomment-2830503
 */
export function useDeepCompareMemo<T>(effect: () => T, deps: React.DependencyList): T {
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(effect, useDeepCompareMemoize(deps))
}
