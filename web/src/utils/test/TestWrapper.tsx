import React, { PropsWithChildren } from 'react'
import QueryString from 'qs'
import { useMemo } from 'react'
import { noop } from 'lodash-es'
import { compile } from 'path-to-regexp'
import { RestfulProvider } from 'restful-react'
import { IconoirProvider } from 'iconoir-react'
import { Route, Router, Switch } from 'react-router-dom'
import { createMemoryHistory } from 'history'

import { routes } from 'RouteDefinitions'
import { ModalProvider } from 'hooks/useModalHook'
import type { UseStringsReturn } from 'framework/strings'
import { CurrentLocation } from 'utils/test/testUtils'
import type { LangLocale } from 'framework/strings/languageLoader'
import { AppContextProvider, defaultCurrentUser } from 'AppContext'
import { defaultUsefulOrNot } from 'components/DefaultUsefulOrNot/UsefulOrNot'
import { StringsContextProvider } from 'framework/strings/StringsContextProvider'

export interface TestWrapperProps {
  path?: string
  pathParams?: Record<string, string | number>
  queryParams?: Record<string, unknown>
  stringsData?: Record<string, string>
  getString?: UseStringsReturn['getString']
  standalone?: boolean
  routingId?: string
  space?: string
  lang?: LangLocale
  currentUser?: typeof defaultCurrentUser
  currentUserProfileURL?: string
  defaultSettingsURL?: string
  isPublicAccessEnabledOnResources?: boolean
  isCurrentSessionPublic?: boolean
  accountInfo?: Unknown
}

export default function TestWrapper(props: PropsWithChildren<TestWrapperProps>) {
  const {
    queryParams,
    path = '/',
    pathParams,
    stringsData = {},
    getString = (key: string) => key,
    standalone = false,
    routingId = '',
    space = '',
    lang = 'en',
    currentUser = defaultCurrentUser,
    currentUserProfileURL = '',
    defaultSettingsURL = '',
    isPublicAccessEnabledOnResources = false,
    isCurrentSessionPublic = false,
    accountInfo = null
  } = props
  const search = QueryString.stringify(queryParams, { addQueryPrefix: true })
  const routePath = compile(path)(pathParams) + search
  const history = useMemo(() => createMemoryHistory({ initialEntries: [routePath] }), [])

  return (
    <Router history={history}>
      <AppContextProvider
        value={{
          standalone,
          routingId,
          space,
          routes,
          lang,
          on401: noop,
          hooks: {},
          currentUser,
          customComponents: {
            UsefulOrNot: defaultUsefulOrNot
          },
          currentUserProfileURL,
          defaultSettingsURL,
          isPublicAccessEnabledOnResources,
          isCurrentSessionPublic,
          accountInfo
        }}>
        <StringsContextProvider initialStrings={stringsData} getString={getString}>
          <RestfulProvider base="/">
            <IconoirProvider
              iconProps={{
                strokeWidth: 1.5,
                width: '16px',
                height: '16px'
              }}>
              <ModalProvider>
                <Switch>
                  <Route exact path={path}>
                    {props.children}
                  </Route>
                  <Route>
                    <CurrentLocation />
                  </Route>
                </Switch>
              </ModalProvider>
            </IconoirProvider>
          </RestfulProvider>
        </StringsContextProvider>
      </AppContextProvider>
    </Router>
  )
}
