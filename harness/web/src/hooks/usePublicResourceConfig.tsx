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

import { useEffect, useMemo } from 'react'
import { useGet } from 'restful-react'
import { useAppContext } from 'AppContext'

export function usePublicResourceConfig() {
  const { standalone, hooks } = useAppContext()
  const { refetchAuthSettings, authSettings, fetchingAuthSettings, errorWhileFetchingAuthSettings } =
    hooks.useGetAuthSettings()
  const {
    data: systemConfig,
    loading: systemConfigLoading,
    error: systemConfigError
  } = useGet({ path: 'api/v1/system/config' })

  useEffect(() => {
    if (!standalone) refetchAuthSettings()
  }, [refetchAuthSettings])

  const allowPublicResourceCreation = useMemo(() => {
    if (systemConfigLoading || fetchingAuthSettings) {
      return false
    }
    if (standalone) {
      return systemConfig?.public_resource_creation_enabled
    }
    return !!(systemConfig?.public_resource_creation_enabled && authSettings?.resource?.publicAccessEnabled)
  }, [authSettings, systemConfig, standalone, systemConfigLoading, fetchingAuthSettings])

  const UIFlags = useMemo(() => {
    if (systemConfigLoading) {
      return false
    }
    if (standalone) {
      return systemConfig?.ui
    }
    return false
  }, [systemConfig, standalone, systemConfigLoading])

  return {
    allowPublicResourceCreation,
    configLoading: fetchingAuthSettings || systemConfigLoading,
    systemConfigError,
    errorWhileFetchingAuthSettings,
    UIFlags
  }
}
