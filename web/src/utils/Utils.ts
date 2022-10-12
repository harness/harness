import { Intent, IToaster, IToastProps, Position, Toaster } from '@blueprintjs/core'
import type { editor as EDITOR } from 'monaco-editor/esm/vs/editor/editor.api'
import { get } from 'lodash-es'
import moment from 'moment'
import { useEffect } from 'react'
import { useAppContext } from 'AppContext'

export const LIST_FETCHING_PAGE_SIZE = 20
export const DEFAULT_DATE_FORMAT = 'MM/DD/YYYY hh:mm a'
export const X_TOTAL = 'x-total'
export const X_TOTAL_PAGES = 'x-total-pages'
export const X_PER_PAGE = 'x-per-page'
export type Unknown = any // eslint-disable-line @typescript-eslint/no-explicit-any
export const DEFAULT_BRANCH_NAME = 'main'
export const REGEX_VALID_REPO_NAME = /^[A-Za-z0-9_.-][A-Za-z0-9 _.-]*$/
export const SUGGESTED_BRANCH_NAMES = ['main', 'master']

/** This utility shows a toaster without being bound to any component.
 * It's useful to show cross-page/component messages */
export function showToaster(message: string, props?: Partial<IToastProps>): IToaster {
  const toaster = Toaster.create({ position: Position.TOP })
  toaster.show({ message, intent: Intent.SUCCESS, ...props })
  return toaster
}

// eslint-disable-next-line
export const getErrorMessage = (error: any): string =>
  get(error, 'data.error', get(error, 'data.message', error?.message))

export const MonacoEditorOptions = {
  ignoreTrimWhitespace: true,
  minimap: { enabled: false },
  codeLens: false,
  scrollBeyondLastLine: false,
  smartSelect: false,
  tabSize: 2,
  insertSpaces: true,
  overviewRulerBorder: false
}

export const MonacoEditorJsonOptions = {
  ...MonacoEditorOptions,
  tabSize: 2
}

// Monaco editor has a bug where when its value is set, the value
// is selected all by default.
// Fix by set selection range to zero
export const deselectAllMonacoEditor = (editor?: EDITOR.IStandaloneCodeEditor): void => {
  editor?.focus()
  setTimeout(() => {
    editor?.setSelection(new monaco.Selection(0, 0, 0, 0))
  }, 0)
}

export const displayDateTime = (value: number): string | null => {
  return value ? moment.unix(value / 1000).format(DEFAULT_DATE_FORMAT) : null
}

const LOCALE = Intl.NumberFormat().resolvedOptions?.().locale || 'en-US'

/**
 * Format a timestamp to short format time (i.e: 7:41 AM)
 * @param timestamp Timestamp
 * @param timeStyle Optional DateTimeFormat's `timeStyle` option.
 */
export function formatTime(timestamp: number, timeStyle = 'short'): string {
  return new Intl.DateTimeFormat(LOCALE, {
    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    // @ts-ignore: TS built-in type for DateTimeFormat is not correct
    timeStyle
  }).format(new Date(timestamp))
}

/**
 * Format a timestamp to medium format date (i.e: Jan 1, 2021)
 * @param timestamp Timestamp
 * @param dateStyle Optional DateTimeFormat's `dateStyle` option.
 */
export function formatDate(timestamp: number, dateStyle = 'medium'): string {
  return new Intl.DateTimeFormat(LOCALE, {
    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    // @ts-ignore: TS built-in type for DateTimeFormat is not correct
    dateStyle
  }).format(new Date(timestamp))
}

export enum Editions {
  ENTERPRISE = 'ENTERPRISE',
  TEAM = 'TEAM',
  FREE = 'FREE',
  COMMUNITY = 'COMMUNITY'
}

export interface License {
  accountIdentifier?: string
  createdAt?: number
  edition?: 'COMMUNITY' | 'FREE' | 'TEAM' | 'ENTERPRISE'
  expiryTime?: number
  id?: string
  lastModifiedAt?: number
  licenseType?: 'TRIAL' | 'PAID'
  moduleType?: 'CD' | 'CI' | 'CV' | 'CF' | 'CE' | 'STO' | 'CORE' | 'PMS' | 'TEMPLATESERVICE' | 'GOVERNANCE'
  premiumSupport?: boolean
  selfService?: boolean
  startTime?: number
  status?: 'ACTIVE' | 'DELETED' | 'EXPIRED'
  trialExtended?: boolean
}

export interface LicenseInformation {
  [key: string]: License
}

export const findEnterprisePaid = (licenseInformation: LicenseInformation): boolean => {
  return !!Object.values(licenseInformation).find(
    (license: License) => license.edition === Editions.ENTERPRISE && license.licenseType === 'PAID'
  )
}

export const useAnyTrialLicense = (): boolean => {
  const {
    hooks: { useLicenseStore = () => ({}) }
  } = useAppContext()
  const { licenseInformation }: { licenseInformation: LicenseInformation } = useLicenseStore()

  const hasEnterprisePaid = findEnterprisePaid(licenseInformation)
  if (hasEnterprisePaid) return false

  const anyTrialEntitlements = Object.values(licenseInformation).find(
    (license: License) => license?.edition === Editions.ENTERPRISE && license?.licenseType === 'TRIAL'
  )

  return !!anyTrialEntitlements
}

export const useGetTrialInfo = (): Unknown => {
  const {
    hooks: { useLicenseStore = () => ({}) }
  } = useAppContext()
  const { licenseInformation }: { licenseInformation: LicenseInformation } = useLicenseStore()

  const hasEnterprisePaid = findEnterprisePaid(licenseInformation)
  if (hasEnterprisePaid) return

  const allEntitlements = Object.keys(licenseInformation).map(module => {
    return licenseInformation[module]
  })

  const trialEntitlement = allEntitlements
    .sort((a: License, b: License) => (b.expiryTime ?? 0) - (a.expiryTime ?? 0))
    .find((license: License) => license?.edition === Editions.ENTERPRISE && license?.licenseType === 'TRIAL')

  return trialEntitlement
}

export const useFindActiveEnterprise = (): boolean => {
  const {
    hooks: { useLicenseStore = () => ({}) }
  } = useAppContext()
  const { licenseInformation }: { licenseInformation: LicenseInformation } = useLicenseStore()
  return Object.values(licenseInformation).some(
    (license: License) => license.edition === Editions.ENTERPRISE && license.status === 'ACTIVE'
  )
}

/**
 * Scrolls the target element to top when any dependency changes
 * @param {string} target Target element className selector
 * @param {array} dependencies Dependencies to watch
 * @returns {void}
 */
export const useScrollToTop = (target: string, dependencies: unknown[]): void => {
  useEffect(() => {
    const element = document.querySelector(`.${target}`)
    if (element) {
      element.scrollTop = 0
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dependencies])
}

/**
 * Make any HTML element as a clickable button with keyboard accessibility
 * support (hit Enter/Space will trigger click event)
 */
export const ButtonRoleProps = {
  onKeyDown: (e: React.KeyboardEvent<HTMLDivElement>): void => {
    if (e.key === 'Enter' || e.key === ' ' || e.key === 'Spacebar' || e.which === 13 || e.which === 32) {
      ;(e.target as unknown as { click: () => void })?.click?.()
    }
  },
  tabIndex: 0,
  role: 'button'
}
