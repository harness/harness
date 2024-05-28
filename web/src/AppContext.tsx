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

import { Button } from '@harnessio/uicore'
import React, { useState, useContext, useEffect, useMemo, createContext } from 'react'
import { matchPath } from 'react-router-dom'
import { useAtom } from 'jotai'
import { noop, merge } from 'lodash-es'
import { useGet } from 'restful-react'
import type { AppProps } from 'AppProps'
import { routes } from 'RouteDefinitions'
import type { TypesUser } from 'services/code'
import { currentUserAtom } from 'atoms/currentUser'
import { newCacheStrategy } from 'utils/CacheStrategy'
import { useGetSettingValue } from 'hooks/useGetSettingValue'
import { useFeatureFlags } from 'hooks/useFeatureFlag'
import { defaultUsefulOrNot } from 'components/DefaultUsefulOrNot/UsefulOrNot'
import type {
  AppStoreContextProps,
  CommonComponents,
  Hooks,
  LicenseStoreContextProps,
  ParentContext,
  PermissionsContextProps,
} from '@harness/microfrontends'
import type {
  Title,
  UseDocumentTitleReturn,
} from '@harness/microfrontends/dist/modules/10-common/hooks/useDocumentTitle'
import type {
  UseCreateConnectorModalProps,
  UseCreateConnectorModalReturn,
} from '@harness/microfrontends/dist/modules/27-platform/connectors/modals/ConnectorModal/useCreateConnectorModal'
import type { ConnectorModaldata } from '@harness/microfrontends/dist/modules/27-platform/connectors/interfaces/ConnectorInterface'
import type { UseLogoutReturn } from '@harness/microfrontends/dist/framework/utils/SessionUtils'
import type { PageParams, TelemetryReturnType } from '@harness/microfrontends/dist/modules/10-common/hooks/useTelemetry'
import type { PermissionsRequest } from '@harness/microfrontends/dist/modules/20-rbac/hooks/usePermission'
import type {
  RBACError,
  RbacErrorReturn,
} from '@harness/microfrontends/dist/modules/20-rbac/utils/useRBACError/useRBACError'
import type {
  ConnectorInfoDTO,
} from '@harness/microfrontends/dist/services/cd-ng'
import type {
  CheckFeatureReturn,
  FeatureRequest,
  FeatureRequestOptions,
} from '@harness/microfrontends/dist/framework/featureStore/featureStoreUtil'
import type { IDialogProps } from '@blueprintjs/core'

interface AppContextProps extends AppProps {
  setAppContext: (value: Partial<AppProps>) => void
}

export const defaultCurrentUser: Required<TypesUser> = {
  admin: false,
  blocked: false,
  created: 0,
  updated: 0,
  display_name: '',
  email: '',
  uid: ''
}

export const stubCommonComponents: CommonComponents = {
  RbacButton: Button,
  RbacMenuItem: () => <div data-testid="rbac-menu-item" />,
  NGBreadcrumbs: () => <div data-testid="ng-breadcrumbs" />,
  YAMLBuilder: () => <div data-testid="yaml-builder" />,
  MonacoEditor: React.forwardRef(() => <div data-testid="monaco-editor" />), // eslint-disable-line react/display-name
  MonacoDiffEditor: React.forwardRef(() => <div data-testid="monaco-diff-editor" />), // eslint-disable-line react/display-name
}

interface FeatureProps {
  featureRequest?: FeatureRequest
  options?: FeatureRequestOptions
}

/* eslint-disable no-console */
export const stubHooks: Required<Hooks> = {
  useDocumentTitle: (_: Title): UseDocumentTitleReturn => ({
    updateTitle: (__: Title): void => console.warn('stub updateTitle() called'),
  }),
  useTelemetry: (_pageParams?: PageParams): TelemetryReturnType => {
    console.warn('stub useTelemetry() called')
    return {} as TelemetryReturnType
  },
  useLogout: (): UseLogoutReturn => ({ forceLogout: () => console.warn('stub forceLogout() called') }),
  useRBACError: (): RbacErrorReturn => ({
    getRBACErrorMessage: (_: RBACError): React.ReactElement | string => {
      console.warn('stub getRBACErrorMessage() called')
      return 'Fake RBAC error message'
    },
  }),
  usePermission: (_?: PermissionsRequest, __?: Array<unknown>): Array<boolean> => {
    console.warn('stub usePermission() called')
    return []
  },
  useCreateConnectorModal: (_: UseCreateConnectorModalProps): UseCreateConnectorModalReturn => ({
    openConnectorModal: (
      __: boolean,
      ___: ConnectorInfoDTO['type'],
      ____?: ConnectorModaldata,
      _____?: IDialogProps,
    ): void => console.warn('stub openConnectorModal() called'),
    hideConnectorModal: (): void => console.warn('stub hideConnectorModal() called'),
  }),
  useFeature: (_props: FeatureProps): CheckFeatureReturn => {
    console.warn('stub useFeature() called')
    return { enabled: true }
  },
  useEventSourceListener: _ => {
    console.warn('stub useEventSourceListener() called')
    return {
      startListening: () => {
        /* No-op */
      },
      stopListening: () => {
        /* No-op */
      },
    }
  },
}


const AppContext = React.createContext<AppContextProps>({
  standalone: true,
  setAppContext: noop,
  routes,
  components: stubCommonComponents,
  hooks: stubHooks,
  currentUser: defaultCurrentUser,
  customComponents: {
    UsefulOrNot: defaultUsefulOrNot
  },
  currentUserProfileURL: '',
  routingId: '',
  defaultSettingsURL: '',
  isPublicAccessEnabledOnResources: false,
  isCurrentSessionPublic: false,
  parentContextObj: {} as ParentContext
})

export const AppContextProvider: React.FC<{ value: AppProps }> = React.memo(function AppContextProvider({
  value: initialValue,
  children
}) {
  const lazy = useMemo(
    () => initialValue.standalone && !!matchPath(location.pathname, { path: '/(signin|register)' }),
    [initialValue.standalone]
  )
  const { data: _currentUser, refetch: fetchCurrentUser } = useGet({
    path: '/api/v1/user',
    lazy: true
  })
  const [currentUser, setCurrentUser] = useAtom(currentUserAtom)
  const [appStates, setAppStates] = useState<AppProps>(
    merge({ hooks: { useFeatureFlags, useGetSettingValue } }, initialValue)
  )

  useEffect(() => {
    // Fetch current user when conditions to fetch it matched and
    //  - cache does not exist yet
    //  - or cache is expired
    if (!lazy && (!currentUser || cacheStrategy.isExpired())) {
      fetchCurrentUser()
    }
  }, [lazy, fetchCurrentUser, currentUser])

  useEffect(() => {
    if (_currentUser) {
      setCurrentUser(_currentUser)
      cacheStrategy.update()
    }
  }, [_currentUser, setCurrentUser])

  useEffect(() => {
    if (initialValue.space && initialValue.space !== appStates.space) {
      setAppStates({ ...appStates, ...initialValue })
    }
  }, [initialValue, appStates])

  return (
    <AppContext.Provider
      value={{
        ...appStates,
        currentUser: (currentUser || defaultCurrentUser) as Required<TypesUser>,
        setAppContext: props => {
          setAppStates({ ...appStates, ...props })
        }
      }}>
      {children}
    </AppContext.Provider>
  )
})

export const useAppContext: () => AppContextProps = () => useContext(AppContext)

export const useAppStoreContext: () => AppStoreContextProps = () => {
  const { parentContextObj } = useAppContext()
  return useContext(parentContextObj.appStoreContext)
}

export const useLicenseStoreContext: () => LicenseStoreContextProps = () => {
  const { parentContextObj } = useAppContext()
  return useContext(parentContextObj.licenseStoreProvider)
}

export const usePermissionsContext: () => PermissionsContextProps = () => {
  const { parentContextObj } = useAppContext()
  return useContext(parentContextObj.permissionsContext)
}

const stubTooltipContext = createContext<Record<string, unknown>>({})

export const useTooltipContext: () => Record<string, unknown> = () => {
  // const { parentContextObj } = useAppContext()
  return useContext(/*parentContextObj.tooltipContext ||*/ stubTooltipContext)
}

// export const useCommonComponents: () => CommonComponents = () => {
//   const { components, customComponents } = useAppContext()
//   return Object.assign({}, stubCommonComponents, stubCustomComponents, components, customComponents)
// }
//
// export const useCommonHooks: () => Required<Hooks> = () => {
//   const { hooks, customHooks } = useAppContext()
//   return Object.assign({}, stubHooks, stubCustomHooks, hooks, customHooks)
// }

const cacheStrategy = newCacheStrategy()
