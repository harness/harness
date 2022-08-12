import React, { useState, useContext } from 'react'
import { noop } from 'lodash-es'
import type { AppProps } from 'AppProps'

interface AppContextProps extends AppProps {
  setAppContext: (value: Partial<AppProps>) => void
}

const AppContext = React.createContext<AppContextProps>({
  standalone: true,
  setAppContext: noop,
  hooks: {},
  components: {}
})

export const AppContextProvider: React.FC<{ value: AppProps }> = React.memo(({ value: initialValue, children }) => {
  const [appStates, setAppStates] = useState<AppProps>(initialValue)

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
