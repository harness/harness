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

import { useMemo } from 'react'
import { mapValues } from 'lodash-es'

import { encodePathParams, normalizePath } from '@ar/routes/utils'
import { ARRouteDefinitionsReturn, routeDefinitions } from '@ar/routes/RouteDefinitions'

import { useAppStore } from './useAppStore'
import { useParentUtils } from './useParentUtils'
import { useDecodedParams } from './useDecodedParams'

export function useRoutes(isRouteDestinationRendering = false): ARRouteDefinitionsReturn {
  const { baseUrl, matchPath } = useAppStore()
  const prefixUrl = isRouteDestinationRendering ? matchPath : baseUrl
  const routeParams = useDecodedParams<Record<string, string>>()
  const { getRouteDefinitions } = useParentUtils()
  const transformedRouteDefinitions: ARRouteDefinitionsReturn = useMemo(() => {
    const finalRouteDefinitions =
      typeof getRouteDefinitions === 'function' ? getRouteDefinitions(routeParams) : routeDefinitions
    return mapValues(finalRouteDefinitions, route => (params: any = {}) => {
      const transformedParams: any = Object.keys(params).reduce((acc, curr) => {
        return { ...acc, [curr]: encodePathParams(params[curr]) }
      }, {})
      return normalizePath(`${prefixUrl}/${route(transformedParams)}`)
    })
  }, [prefixUrl])

  return transformedRouteDefinitions
}
