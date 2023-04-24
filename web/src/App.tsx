import React, { useEffect, useState, useCallback, useMemo } from 'react'
import { RestfulProvider } from 'restful-react'
import { Container, TooltipContextProvider } from '@harness/uicore'
import { ModalProvider } from '@harness/use-modal'
import { FocusStyleManager } from '@blueprintjs/core'
import { tooltipDictionary } from '@harness/ng-tooltip'
import AppErrorBoundary from 'framework/AppErrorBoundary/AppErrorBoundary'
import { AppContextProvider, defaultCurrentUser } from 'AppContext'
import type { AppProps } from 'AppProps'
import { buildResfulReactRequestOptions, handle401 } from 'AppUtils'
import { RouteDestinations } from 'RouteDestinations'
import { useAPIToken } from 'hooks/useAPIToken'
import { routes as _routes } from 'RouteDefinitions'
import { getConfig } from 'services/config'
import { languageLoader } from './framework/strings/languageLoader'
import type { LanguageRecord } from './framework/strings/languageLoader'
import { StringsContextProvider } from './framework/strings/StringsContextProvider'
import 'highlight.js/styles/github.css'
import 'diff2html/bundles/css/diff2html.min.css'
import css from './App.module.scss'

FocusStyleManager.onlyShowFocusOnTabs()

const App: React.FC<AppProps> = React.memo(function App({
  standalone = false,
  space = '',
  routes = _routes,
  lang = 'en',
  on401 = handle401,
  children,
  hooks,
  currentUserProfileURL = ''
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

  const Wrapper: React.FC = useCallback(
    props => {
      return strings ? (
        <Container className={css.main}>
          <StringsContextProvider initialStrings={strings}>
            <AppErrorBoundary>
              <RestfulProvider
                base={standalone ? '/' : getConfig('code')}
                requestOptions={getRequestOptions}
                queryParams={queryParams}
                queryParamStringifyOptions={{ skipNulls: true }}
                onResponse={response => {
                  if (!response.ok && response.status === 401) {
                    on401()
                  }
                }}>
                <AppContextProvider
                  value={{
                    standalone,
                    space,
                    routes,
                    lang,
                    on401,
                    hooks,
                    currentUser: defaultCurrentUser,
                    currentUserProfileURL
                  }}>
                  <TooltipContextProvider initialTooltipDictionary={tooltipDictionary}>
                    <ModalProvider>{props.children ? props.children : <RouteDestinations />}</ModalProvider>
                  </TooltipContextProvider>
                </AppContextProvider>
              </RestfulProvider>
            </AppErrorBoundary>
          </StringsContextProvider>
        </Container>
      ) : null
    },
    [strings] // eslint-disable-line react-hooks/exhaustive-deps
  )

  useEffect(() => {
    AppWrapper = Wrapper
  }, [Wrapper])

  return <Wrapper>{children}</Wrapper>
})

export let AppWrapper: React.FC = () => <Container />
export default App
