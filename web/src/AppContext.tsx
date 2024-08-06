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

import React, { useState, useContext, useEffect, useMemo } from 'react'
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

const AppContext = React.createContext<AppContextProps>({
  standalone: true,
  setAppContext: noop,
  routes,
  hooks: {},
  currentUser: defaultCurrentUser,
  customComponents: {
    UsefulOrNot: defaultUsefulOrNot
  },
  currentUserProfileURL: '',
  routingId: '',
  defaultSettingsURL: '',
  isPublicAccessEnabledOnResources: false,
  isCurrentSessionPublic: !!window.publicAccessOnGitness
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
    //  - currentSession is not Public
    if (!lazy && (!currentUser || cacheStrategy.isExpired()) && !initialValue.isCurrentSessionPublic) {
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

const cacheStrategy = newCacheStrategy()
