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

import React, { useEffect, useState, useCallback, useMemo } from 'react'
import { RestfulProvider } from 'restful-react'
import { IconoirProvider } from 'iconoir-react'
import cx from 'classnames'
import { Container } from '@harnessio/uicore'
import { FocusStyleManager } from '@blueprintjs/core'
import AppErrorBoundary from 'framework/AppErrorBoundary/AppErrorBoundary'
import { AppContextProvider, defaultCurrentUser } from 'AppContext'
import type { AppProps } from 'AppProps'
import { buildRestfulReactRequestOptions, handle401 } from 'AppUtils'
import { RouteDestinations } from 'RouteDestinations'
import { routes as _routes } from 'RouteDefinitions'
import { getConfig } from 'services/config'
import { ModalProvider } from 'hooks/useModalHook'
import { languageLoader } from './framework/strings/languageLoader'
import type { LanguageRecord } from './framework/strings/languageLoader'
import { StringsContextProvider } from './framework/strings/StringsContextProvider'
import 'highlight.js/styles/github.css'
import 'diff2html/bundles/css/diff2html.min.css'
import './App.scss'
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
  customComponents,
  currentUserProfileURL = '',
  defaultSettingsURL = '',
  isPublicAccessEnabledOnResources = false,
  isCurrentSessionPublic = !!window.publicAccessOnGitness
}: AppProps) {
  const [strings, setStrings] = useState<LanguageRecord>()
  const getRequestOptions = useCallback(
    (): Partial<RequestInit> => buildRestfulReactRequestOptions(hooks?.useGetToken?.() || ''),
    [hooks]
  )
  const routingId = useMemo(() => (standalone ? '' : space.split('/').shift() || ''), [standalone, space])
  const queryParams = useMemo(() => (!standalone ? { routingId } : {}), [standalone, routingId])

  useEffect(() => {
    languageLoader(lang).then(setStrings)
  }, [lang, setStrings])

  const Wrapper: React.FC<{ fullPage: boolean }> = useCallback(
    props => {
      return strings ? (
        <Container className={cx(css.main, { [css.fullPage]: standalone && props.fullPage })}>
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
                    routingId,
                    space,
                    routes,
                    lang,
                    on401,
                    hooks,
                    currentUser: defaultCurrentUser,
                    customComponents,
                    currentUserProfileURL,
                    defaultSettingsURL,
                    isPublicAccessEnabledOnResources,
                    isCurrentSessionPublic
                  }}>
                  <IconoirProvider
                    iconProps={{
                      strokeWidth: 1.5,
                      width: '16px',
                      height: '16px'
                    }}>
                    <ModalProvider>{props.children ? props.children : <RouteDestinations />}</ModalProvider>
                  </IconoirProvider>
                </AppContextProvider>
              </RestfulProvider>
            </AppErrorBoundary>
          </StringsContextProvider>
        </Container>
      ) : null
    },
    [strings, space] // eslint-disable-line react-hooks/exhaustive-deps
  )

  useEffect(() => {
    AppWrapper = function _AppWrapper({ children: _children }) {
      return <Wrapper fullPage={false}>{_children}</Wrapper>
    }
  }, [Wrapper])

  return <Wrapper fullPage>{children}</Wrapper>
})

export let AppWrapper: React.FC = () => <Container />
export default App
