import React, { useState, useContext, useEffect } from 'react'
import { noop } from 'lodash-es'
import { useGet } from 'restful-react'
import type { AppProps } from 'AppProps'
import { routes } from 'RouteDefinitions'
import type { TypesUser } from 'services/code'
import { useAPIToken } from 'hooks/useAPIToken'

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
  const [token, setToken] = useAPIToken()
  const { data: currentUser = defaultCurrentUser, error } = useGet({
    path: '/api/v1/user',
    lazy: initialValue.standalone && !token
  })
  const [appStates, setAppStates] = useState<AppProps>(initialValue)

  useEffect(() => {
    if (initialValue.space && initialValue.space !== appStates.space) {
      setAppStates({ ...appStates, ...initialValue })
    }
  }, [initialValue, appStates])

  useEffect(() => {
    if (initialValue.standalone && error) {
      setToken('')
    }
  }, [initialValue.standalone, error, setToken])

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
