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

import React from 'react'
import ReactDOM from 'react-dom'
import { noop } from 'lodash-es'
import { routes } from 'RouteDefinitions'
import { defaultCurrentUser } from 'AppContext'
import { useFeatureFlags } from 'hooks/useFeatureFlag'
import { useGetSettingValue } from 'hooks/useGetSettingValue'
import { useGetAuthSettings } from 'hooks/useGetAuthSettings'
import { defaultUsefulOrNot } from 'components/DefaultUsefulOrNot/UsefulOrNot'
import { defaultDelegateSelectorsV2 } from 'components/DelegateSelector/DelegateSelector'
import { defaultMultiTypeConnectorField } from 'components/FormMultiTypeConnectorField/FormMultiTypeConnectorField'
import App from './App'
import './bootstrap.scss'

// This flag is used in services/config.ts to customize API path when app is run
// in multiple modes (standalone vs. embedded).
// Also being used in when generating proper URLs inside the app.
// In standalone mode, we don't need `code/` prefix in API URIs.
window.STRIP_CODE_PREFIX = true

ReactDOM.render(
  <App
    standalone
    routes={routes}
    hooks={{
      usePermissionTranslate: noop,
      useExecutionDataHook: noop,
      useLogsContent: noop,
      useLogsStreaming: noop,
      useFeatureFlags,
      useGetSettingValue,
      useGetAuthSettings
    }}
    currentUser={defaultCurrentUser}
    customComponents={{
      UsefulOrNot: defaultUsefulOrNot,
      DelegateSelectorsV2: defaultDelegateSelectorsV2,
      MultiTypeConnectorField: defaultMultiTypeConnectorField
    }}
    currentUserProfileURL=""
    routingId=""
    defaultSettingsURL=""
    isPublicAccessEnabledOnResources
    isCurrentSessionPublic={!!window.publicAccessOnGitness}
    accountInfo={noop}
  />,
  document.getElementById('react-root')
)
