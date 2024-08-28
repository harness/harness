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

import type React from 'react'
import type { CODERoutes } from 'RouteDefinitions'
import type { TypesUser } from 'services/code'
import type { UsefulOrNotProps } from 'utils/types'
import type { LangLocale } from './framework/strings/languageLoader'

/**
 * AppProps defines an interface for host (parent) and
 * child (micro-frontend) apps to talk to each other. It allows behaviors
 * of the child app to be customized from the parent app.
 *
 * Areas of customization:
 *  - API token
 *  - Active user
 *  - Active locale (i18n)
 *  - Global error handling (like 401)
 *  - etc...
 *
 * Under standalone mode, the micro-frontend app uses default
 * implementation of the interface in AppUtils.ts.
 *
 * This interface is published to allow parent to do type checking.
 */
export interface AppProps {
  /** Flag to tell if App is mounted as a standalone app */
  standalone: boolean

  /** Harness routingId */
  routingId: string

  /** App children. When provided, children is a remote view which will be mounted under App contexts */
  children?: React.ReactNode

  /** Active space when app is embedded */
  space?: string

  /** Routing utlis (used to generate app specific URLs) */
  routes: CODERoutes

  /** Language to use in the app, default is 'en' */
  lang?: LangLocale

  /** 401 handler. Used in parent app to override 401 handling from child app */
  on401?: () => void
  /** React Components which are passed down from the Parent that are needed by the child app */
  customComponents: {
    UsefulOrNot: (props: UsefulOrNotProps) => React.ReactElement
  }
  /** React Hooks that Harness Platform passes down. Note: Pass only hooks that your app need */
  hooks: Partial<{
    useGetToken: Unknown
    usePermissionTranslate: Unknown
    useGenerateToken: Unknown
    useExecutionDataHook: Unknown
    useLogsContent: Unknown
    useLogsStreaming: Unknown
    useFeatureFlags: Unknown
    useGetSettingValue: Unknown
    useGetAuthSettings: Unknown
    useGetUserSourceCodeManagers?: Unknown
    useListAggregatedTokens?: Unknown
    useDeleteToken?: Unknown
    useCreateToken?: Unknown
  }>

  currentUser: Required<TypesUser>

  currentUserProfileURL: string
  defaultSettingsURL: string
  isPublicAccessEnabledOnResources: boolean
  isCurrentSessionPublic: boolean
  module?: string

  arAppStore?: {
    repositoryIdentifier?: string
    artifactIdentifier?: string
    versionIdentifier?: string
  }
}
