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
import { noop } from 'lodash-es'
import { useGet } from 'restful-react'
import type { AppProps } from 'AppProps'
import { routes } from 'RouteDefinitions'
import type { TypesUser } from 'services/code'

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
  currentUserProfileURL: '',
  routingId: ''
})

export const AppContextProvider: React.FC<{ value: AppProps }> = React.memo(function AppContextProvider({
  value: initialValue,
  children
}) {
  const lazy = useMemo(
    () => initialValue.standalone && !!matchPath(location.pathname, { path: '/(signin|register)' }),
    [initialValue.standalone]
  )
  const { data: currentUser = defaultCurrentUser } = useGet({
    path: '/api/v1/user',
    lazy
  })
  const [appStates, setAppStates] = useState<AppProps>(initialValue)

  useEffect(() => {
    if (initialValue.space && initialValue.space !== appStates.space) {
      setAppStates({ ...appStates, ...initialValue })
    }
  }, [initialValue, appStates])

  return (
    <AppContext.Provider
      value={{
        ...appStates,
        currentUser: currentUser as Required<TypesUser>,
        setAppContext: props => {
          setAppStates({ ...appStates, ...props })
        }
      }}>
      {children}
    </AppContext.Provider>
  )
})

export const useAppContext: () => AppContextProps = () => useContext(AppContext)
