import type React from 'react'
import type { CODERoutes } from 'RouteDefinitions'
import type { TypesUser } from 'services/code'
import type { LangLocale } from './framework/strings/languageLoader'

/**
 * AppProps defines an interface for host (parent) and
 * child (micro-frontend) apps to talk to each other. It allows behaviors
 * of the child app to be customized from the parent app.
 *
 * Areas of customization:
 *  - API token
 *  - Active user
 *  - Active locale (i18n)
 *  - Global error handling (like 401)
 *  - etc...
 *
 * Under standalone mode, the micro-frontend app uses default
 * implementation of the interface in AppUtils.ts.
 *
 * This interface is published to allow parent to do type checking.
 */
export interface AppProps {
  /** Flag to tell if App is mounted as a standalone app */
  standalone: boolean

  /** App children. When provided, children is a remote view which will be mounted under App contexts */
  children?: React.ReactNode

  /** Active space when app is embedded */
  space?: string

  /** Routing utlis (used to generate app specific URLs) */
  routes: CODERoutes

  /** Language to use in the app, default is 'en' */
  lang?: LangLocale

  /** 401 handler. Used in parent app to override 401 handling from child app */
  on401?: () => void

  /** React Hooks that Harness Platform passes down. Note: Pass only hooks that your app need */
  hooks: Partial<{
    useGetToken: Unknown
  }>

  currentUser: Required<TypesUser>

  currentUserProfileURL: string
}
