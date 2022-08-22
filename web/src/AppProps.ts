import type React from 'react'
import type * as History from 'history'
import type { PermissionOptionsMenuButtonProps } from 'components/Permissions/PermissionsOptionsMenuButton'
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

  /** Base Route information where app is mounted */
  baseRoutePath?: string

  /** Active account id when app is embedded */
  accountId?: string

  /** Language to use in the app, default is 'en' */
  lang?: LangLocale

  /** API token to be used in Restful React */
  apiToken?: string

  /** 401 handler. Used in parent app to override 401 handling from child app */
  on401?: () => void

  /** React Hooks that Harness Platform passes down. Note: Pass only hooks that your app need */
  hooks: Partial<AppPropsHook>

  /** React Components that Harness Platform passes down. Note: Pass only components that your app need */
  components: Partial<AppPropsComponent>
}

/**
 * AppPathProps defines all possible URL parameters that application accepts.
 */
export interface AppPathProps {
  accountId?: string
  orgIdentifier?: string
  projectIdentifier?: string
  module?: string
  policyIdentifier?: string
  policySetIdentifier?: string
  evaluationId?: string
  repo?: string
  branch?: string
}

/**
 * AppPropsHook defines a collection of React Hooks that application receives from
 * Platform integration.
 */
export interface AppPropsHook {
  usePermission(permissionRequest: any, deps?: Array<any>): Array<boolean>
  useGetSchemaYaml(params: any, deps?: Array<any>): Record<string, any>
  useGetToken(): any
  useAppStore(): any
  useGitSyncStore(): any
  useSaveToGitDialog(props: { onSuccess: any; onClose: any; onProgessOverlayClose: any }): any
  useGetListOfBranchesWithStatus(props: any): any
  useAnyEnterpriseLicense(): boolean
  useCurrentEnterpriseLicense(): boolean
  useLicenseStore(): any
} // eslint-disable-line  @typescript-eslint/no-empty-interface

/**
 * AppPropsComponent defines a collection of React Components that application receives from
 * Platform integration.
 */
export interface AppPropsComponent {
  NGBreadcrumbs: React.FC
  RbacButton: React.FC
  RbacOptionsMenuButton: React.FC<PermissionOptionsMenuButtonProps>
  GitSyncStoreProvider: React.FC
  GitContextForm: React.FC<any>
  NavigationCheck: React.FC<{
    when?: boolean
    textProps?: {
      contentText?: string
      titleText?: string
      confirmButtonText?: string
      cancelButtonText?: string
    }
    navigate: (path: string) => void
    shouldBlockNavigation?: (location: History.Location) => boolean
  }>
}
