import React, { useEffect, useState, useCallback, useMemo } from 'react'
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
import { routes as _routes } from 'RouteDefinitions'
import { getConfigNew } from 'services/config'
import { languageLoader } from './framework/strings/languageLoader'
import type { LanguageRecord } from './framework/strings/languageLoader'
import { StringsContextProvider } from './framework/strings/StringsContextProvider'
import './App.scss'

FocusStyleManager.onlyShowFocusOnTabs()

const App: React.FC<AppProps> = React.memo(function App({
  standalone = false,
  space = '',
  routes = _routes,
  lang = 'en',
  on401 = handle401,
  children,
  hooks
}: AppProps) {
  const [strings, setStrings] = useState<LanguageRecord>()
  const token = useAPIToken()
  const getRequestOptions = useCallback(
    (): Partial<RequestInit> => buildResfulReactRequestOptions(hooks?.useGetToken?.() || token),
    [token, hooks]
  )
  const queryParams = useMemo(() => (!standalone ? { routingId: space.split('/').shift() } : {}), [space, standalone])

  useEffect(() => {
    languageLoader(lang).then(setStrings)
  }, [lang, setStrings])

  // Workaround to disable editor dark mode (https://github.com/uiwjs/react-markdown-editor#support-dark-modenight-mode)
  document.documentElement.setAttribute('data-color-mode', 'light')

  return strings ? (
    <StringsContextProvider initialStrings={strings}>
      <AppErrorBoundary>
        <AppContextProvider value={{ standalone, space, routes, lang, on401, hooks }}>
          <RestfulProvider
            base={standalone ? '/' : getConfigNew('code')}
            requestOptions={getRequestOptions}
            queryParams={queryParams}
            queryParamStringifyOptions={{ skipNulls: true }}
            onResponse={response => {
              if (!response.ok && response.status === 401) {
                on401()
              }
            }}>
            <TooltipContextProvider initialTooltipDictionary={tooltipDictionary}>
              <ModalProvider>{children ? children : <RouteDestinations />}</ModalProvider>
            </TooltipContextProvider>
          </RestfulProvider>
        </AppContextProvider>
      </AppErrorBoundary>
    </StringsContextProvider>
  ) : null
})

export default App
