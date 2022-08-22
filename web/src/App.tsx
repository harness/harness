import React, { useEffect, useState, useCallback } from 'react'
import { RestfulProvider } from 'restful-react'
import { TooltipContextProvider } from '@harness/uicore'
import { ModalProvider } from '@harness/use-modal'
import { FocusStyleManager } from '@blueprintjs/core'
import { tooltipDictionary } from '@harness/ng-tooltip'
import AppErrorBoundary from 'framework/AppErrorBoundary/AppErrorBoundary'
import { AppContextProvider } from 'AppContext'
import type { AppProps } from 'AppProps'
import { buildResfulReactRequestOptions, handle401 } from 'AppUtils'
import { RouteDestinations } from 'RouteDestinations'
import { useAPIToken } from 'hooks/useAPIToken'
import { languageLoader } from './framework/strings/languageLoader'
import type { LanguageRecord } from './framework/strings/languageLoader'
import { StringsContextProvider } from './framework/strings/StringsContextProvider'

FocusStyleManager.onlyShowFocusOnTabs()

const App: React.FC<AppProps> = ({
  standalone = false,
  accountId = '',
  lang = 'en',
  apiToken,
  on401 = handle401,
  children,
  hooks = {},
  components = {}
}) => {
  const [strings, setStrings] = useState<LanguageRecord>()
  const [token, setToken] = useAPIToken(apiToken)
  const getRequestOptions = useCallback((): Partial<RequestInit> => {
    return buildResfulReactRequestOptions(hooks.useGetToken?.() || apiToken || 'default')
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

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
        <AppContextProvider value={{ standalone, accountId, lang, on401, hooks, components }}>
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
            <TooltipContextProvider initialTooltipDictionary={tooltipDictionary}>
              <ModalProvider>{children ? children : <RouteDestinations standalone={standalone} />}</ModalProvider>
            </TooltipContextProvider>
          </RestfulProvider>
        </AppContextProvider>
      </AppErrorBoundary>
    </StringsContextProvider>
  ) : null
}

export default App
