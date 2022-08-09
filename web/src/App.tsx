import React, { useEffect, useState, useCallback } from 'react'
import { RestfulProvider } from 'restful-react'
import { TooltipContextProvider, ModalProvider } from '@harness/uicore'
import { FocusStyleManager } from '@blueprintjs/core'
import AppErrorBoundary from 'framework/AppErrorBoundary/AppErrorBoundary'
import { useAPIToken } from 'hooks/useAPIToken'
import { AppContextProvider } from 'AppContext'
import { setBaseRouteInfo } from 'RouteUtils'
import type { AppProps } from 'AppProps'
import { buildResfulReactRequestOptions, handle401 } from 'AppUtils'
import { RouteDestinations } from 'RouteDestinations'
import { languageLoader } from './framework/strings/languageLoader'
import type { LanguageRecord } from './framework/strings/languageLoader'
import { StringsContextProvider } from './framework/strings/StringsContextProvider'
import './App.scss'

FocusStyleManager.onlyShowFocusOnTabs()

const App: React.FC<AppProps> = props => {
  const {
    standalone = false,
    accountId = '',
    baseRoutePath = '',
    lang = 'en',
    apiToken,
    on401 = handle401,
    children,
    hooks = {},
    components = {}
  } = props
  const [strings, setStrings] = useState<LanguageRecord>()
  const [token, setToken] = useAPIToken(apiToken)
  const getRequestOptions = useCallback((): Partial<RequestInit> => {
    return buildResfulReactRequestOptions(token)
  }, [token])
  setBaseRouteInfo(accountId, baseRoutePath)

  useEffect(() => {
    languageLoader(lang).then(setStrings)
  }, [lang, setStrings])

  useEffect(() => {
    if (!apiToken) {
      setToken(token)
    }
  }, [apiToken, token, setToken])

  return strings ? (
    <StringsContextProvider initialStrings={strings}>
      <AppErrorBoundary>
        <AppContextProvider value={{ standalone, baseRoutePath, accountId, lang, apiToken, on401, hooks, components }}>
          <RestfulProvider
            base="/"
            requestOptions={getRequestOptions}
            queryParams={{}}
            queryParamStringifyOptions={{ skipNulls: true }}
            onResponse={response => {
              if (!response.ok && response.status === 401) {
                on401()
              }
            }}>
            <TooltipContextProvider initialTooltipDictionary={{}}>
              <ModalProvider>{children ? children : <RouteDestinations standalone={standalone} />}</ModalProvider>
            </TooltipContextProvider>
          </RestfulProvider>
        </AppContextProvider>
      </AppErrorBoundary>
    </StringsContextProvider>
  ) : null
}

export default App
