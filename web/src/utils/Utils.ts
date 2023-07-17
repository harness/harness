import { Intent, IToaster, IToastProps, Position, Toaster } from '@blueprintjs/core'
import type { editor as EDITOR } from 'monaco-editor/esm/vs/editor/editor.api'
import { get } from 'lodash-es'
import moment from 'moment'
import langMap from 'lang-map'

export const LIST_FETCHING_LIMIT = 20
export const DEFAULT_DATE_FORMAT = 'MM/DD/YYYY hh:mm a'
export const DEFAULT_BRANCH_NAME = 'main'
export const REGEX_VALID_REPO_NAME = /^[a-zA-Z_][0-9a-zA-Z-_.$]*$/
export const SUGGESTED_BRANCH_NAMES = [DEFAULT_BRANCH_NAME, 'master']
export const FILE_SEPERATOR = '/'
export const INITIAL_ZOOM_LEVEL = 1
export const ZOOM_INC_DEC_LEVEL = 0.1

/** This utility shows a toaster without being bound to any component.
 * It's useful to show cross-page/component messages */
export function showToaster(message: string, props?: Partial<IToastProps>): IToaster {
  const toaster = Toaster.create({ position: Position.TOP })
  toaster.show({ message, intent: Intent.SUCCESS, ...props })
  return toaster
}

export function permissionProps(permResult: { disabled: boolean; tooltip: JSX.Element | string }, standalone: boolean) {
  const perm = standalone ? true : false
  return !perm ? permResult : undefined
}

export function generateAlphaNumericHash(length: number) {
  let result = ''
  const characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  const charactersLength = characters.length

  for (let i = 0; i < length; i++) {
    result += characters.charAt(Math.floor(Math.random() * charactersLength))
  }

  return result
}

export const dayAgoInMS = 86400000

export const getErrorMessage = (error: Unknown): string | undefined =>
  error ? get(error, 'data.error', get(error, 'data.message', get(error, 'message', error))) : undefined

export interface PageBrowserProps {
  page: string
}

export interface RenameDetails {
  commit_sha_after: string
  commit_sha_before: string
  new_path: string
  old_path: string
}

export interface LoginForm {
  username: string
  password: string
}

export interface RegisterForm {
  username: string
  password: string
  email: string
}

export interface SourceCodeEditorProps {
  source: string
  language?: string
  lineNumbers?: boolean
  readOnly?: boolean
  highlightLines?: string // i.e: {1,3-4}, TODO: not yet supported
  height?: number | string
  autoHeight?: boolean
  wordWrap?: boolean
  onChange?: (value: string) => void
}

// Monaco editor has a bug where when its value is set, the value
// is selected all by default.
// Fix by set selection range to zero
export const deselectAllMonacoEditor = (editor?: EDITOR.IStandaloneCodeEditor) => {
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
export function formatTime(timestamp: number | string, timeStyle = 'short'): string {
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
export function formatDate(timestamp: number | string, dateStyle = 'medium'): string {
  return new Intl.DateTimeFormat(LOCALE, {
    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    // @ts-ignore: TS built-in type for DateTimeFormat is not correct
    dateStyle
  }).format(new Date(timestamp))
}

/**
 * Format a number with current locale.
 * @param num number
 * @returns Formatted string.
 */
export function formatNumber(num: number | bigint): string {
  return new Intl.NumberFormat(LOCALE).format(num)
}

/**
 * Make any HTML element as a clickable button with keyboard accessibility
 * support (hit Enter/Space will trigger click event)
 */
export const ButtonRoleProps = {
  onKeyDown: (e: React.KeyboardEvent<HTMLElement>) => {
    if (e.key === 'Enter' || e.key === ' ' || e.key === 'Spacebar' || e.which === 13 || e.which === 32) {
      ;(e.target as unknown as { click: () => void })?.click?.()
    }
  },
  tabIndex: 0,
  role: 'button',
  style: { cursor: 'pointer ' }
}

export enum orderSortDate {
  ASC = 'asc',
  DESC = 'desc'
}

export enum CodeCommentState {
  ACTIVE = 'active',
  RESOLVED = 'resolved'
}

export enum FileSection {
  CONTENT = 'content',
  BLAME = 'blame',
  HISTORY = 'history'
}

const MONACO_SUPPORTED_LANGUAGES = [
  'abap',
  'apex',
  'azcli',
  'bat',
  'cameligo',
  'clojure',
  'coffee',
  'cpp',
  'csharp',
  'csp',
  'css',
  'dockerfile',
  'fsharp',
  'go',
  'graphql',
  'handlebars',
  'html',
  'ini',
  'java',
  'javascript',
  'json',
  'kotlin',
  'less',
  'lua',
  'markdown',
  'mips',
  'msdax',
  'mysql',
  'objective-c',
  'pascal',
  'pascaligo',
  'perl',
  'pgsql',
  'php',
  'postiats',
  'powerquery',
  'powershell',
  'pug',
  'python',
  'r',
  'razor',
  'redis',
  'redshift',
  'restructuredtext',
  'ruby',
  'rust',
  'sb',
  'scheme',
  'scss',
  'shell',
  'solidity',
  'sophia',
  'sql',
  'st',
  'swift',
  'tcl',
  'twig',
  'typescript',
  'vb',
  'xml',
  'yaml'
]

const EXTENSION_TO_LANG: Record<string, string> = {
  tsx: 'typescript',
  jsx: 'typescript'
}

export const PLAIN_TEXT = 'plaintext'

export const filenameToLanguage = (name?: string): string | undefined => {
  const extension = name?.split('.').pop() || ''
  const map = langMap.languages(extension)

  if (map?.length) {
    return MONACO_SUPPORTED_LANGUAGES.find(lang => map.includes(lang)) || EXTENSION_TO_LANG[extension] || PLAIN_TEXT
  }

  return PLAIN_TEXT
}

export function waitUntil(condition: () => boolean, callback: () => void, maxCount = 100, timeout = 50) {
  if (condition()) {
    callback()
  } else {
    if (maxCount) {
      setTimeout(() => {
        waitUntil(condition, callback, maxCount - 1)
      }, timeout)
    }
  }
}

/**
 * Make a callback function that will be executed with no arg and return nothing.
 * @param f Any function.
 * @returns A callback function that accept no arguments and return nothing.
 */
// eslint-disable-next-line @typescript-eslint/ban-types
export const voidFn = (f: Function) => () => {
  f()
}

export enum MergeCheckStatus {
  MERGEABLE = 'mergeable',
  UNCHECKED = 'unchecked',
  CONFLICT = 'conflict'
}

/**
 * Convert number of bytes into human readable format
 *
 * @param integer bytes     Number of bytes to convert
 * @param integer precision Number of digits after the decimal separator
 * @return string
 * @link https://stackoverflow.com/a/18650828/1114931
 */
export function formatBytes(bytes: number, decimals = 2) {
  if (!+bytes) return '0 Bytes'

  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ['Bytes', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB', 'EiB', 'ZiB', 'YiB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))

  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`
}
