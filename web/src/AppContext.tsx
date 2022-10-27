import React, { useState, useContext, useEffect } from 'react'
import { noop } from 'lodash-es'
import type { AppProps } from 'AppProps'
import { routes } from 'RouteDefinitions'

interface AppContextProps extends AppProps {
  setAppContext: (value: Partial<AppProps>) => void
}

const AppContext = React.createContext<AppContextProps>({
  standalone: true,
  setAppContext: noop,
  routes,
  hooks: {}
})

export const AppContextProvider: React.FC<{ value: AppProps }> = React.memo(function AppContextProvider({
  value: initialValue,
  children
}) {
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
        setAppContext: props => {
          setAppStates({ ...appStates, ...props })
        }
      }}>
      {children}
    </AppContext.Provider>
  )
})

export const useAppContext: () => AppContextProps = () => useContext(AppContext)
