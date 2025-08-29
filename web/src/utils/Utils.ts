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

import { IconName, Intent, IToaster, IToastProps, Position, Toaster } from '@blueprintjs/core'
import { get } from 'lodash-es'
import moment from 'moment'
import langMap from 'lang-map'
import type { EditorDidMount } from 'react-monaco-editor'
import type { editor } from 'monaco-editor'
import type { EditorView } from '@codemirror/view'
import type { FormikProps } from 'formik'
import type { SelectOption } from '@harnessio/uicore'
import type {
  RepoRepositoryOutput,
  TypesLabel,
  TypesLabelValue,
  TypesPrincipalInfo,
  EnumMembershipRole,
  TypesDefaultReviewerApprovalsResponse,
  TypesUserGroupInfo
} from 'services/code'
import type { StringKeys } from 'framework/strings'
import { PullReqReviewDecision } from 'pages/PullRequest/PullRequestUtils'
import type { Identifier } from './types'
import { CodeIcon } from './GitUtils'

/**
 * Utility to check if the value of the string is 'true'.
 * Useful for query param string checks.
 */
export function isParamTrue(val?: string): boolean {
  return val === 'true'
}

export enum ACCESS_MODES {
  VIEW,
  EDIT
}

export enum ResourceType {
  ACCOUNT = 'ACCOUNT',
  ORGANIZATION = 'ORGANIZATION',
  PROJECT = 'PROJECT',
  CODE_REPOSITORY = 'CODE_REPOSITORY',
  SPACE = 'SPACE'
}

export enum PermissionIdentifier {
  CREATE_PROJECT = 'core_project_create',
  UPDATE_PROJECT = 'core_project_edit',
  DELETE_PROJECT = 'core_project_delete',
  VIEW_PROJECT = 'core_project_view',
  CREATE_ORG = 'core_organization_create',
  UPDATE_ORG = 'core_organization_edit',
  DELETE_ORG = 'core_organization_delete',
  VIEW_ORG = 'core_organization_view',
  CREATE_ACCOUNT = 'core_account_create',
  UPDATE_ACCOUNT = 'core_account_edit',
  DELETE_ACCOUNT = 'core_account_delete',
  VIEW_ACCOUNT = 'core_account_view',
  CODE_REPO_EDIT = 'code_repo_edit',
  CODE_REPO_PUSH = 'code_repo_push',
  CODE_REPO_VIEW = 'code_repo_view',
  CODE_REPO_REVIEW = 'code_repo_review',
  SPACE_VIEW = 'space_view',
  SPACE_EDIT = 'space_edit'
}

export enum PullRequestSection {
  CONVERSATION = 'conversation',
  COMMITS = 'commits',
  FILES_CHANGED = 'changes',
  CHECKS = 'checks'
}

export enum FeatureType {
  COMINGSOON = 'comingSoon',
  RELEASED = 'released'
}

export enum REPO_EXPORT_STATE {
  FINISHED = 'finished',
  FAILED = 'failed',
  CANCELED = 'canceled',
  RUNNING = 'running',
  SCHEDULED = 'scheduled'
}

export enum PrincipalType {
  USER = 'user',
  SERVICE = 'service',
  SERVICE_ACCOUNT = 'serviceaccount',
  USER_GROUP = 'usergroup'
}

export const INCLUDE_INHERITED_GROUPS = 'INCLUDE_INHERITED_GROUPS'
export const LIST_FETCHING_LIMIT = 20
export const DEFAULT_DATE_FORMAT = 'MM/DD/YYYY hh:mm a'
export const DEFAULT_BRANCH_NAME = 'main'
export const REGEX_VALID_REPO_NAME = /^[a-zA-Z_][0-9a-zA-Z-_.$]*$/
export const SUGGESTED_BRANCH_NAMES = [DEFAULT_BRANCH_NAME, 'master']
export const FILE_SEPARATOR = '/'
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
  return standalone ? undefined : permResult
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
  author?: string
  page?: string
  state?: string
  tab?: string
  review?: string
  sort?: string
  only_favorites?: string
  recursive?: string
  inherit?: string
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
  confirmPassword: string
  email: string
}

export interface FeatureData {
  typeText: string
  type: string
  title: string
  content: string
  link: string
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
  schema?: Record<string, unknown>
  editorDidMount?: EditorDidMount
  editorOptions?: editor.IStandaloneEditorConstructionOptions
}

export const displayDateTime = (value: number): string | null => {
  return value ? moment.unix(value / 1000).format(DEFAULT_DATE_FORMAT) : null
}

export const timeDistance = (date1 = 0, date2 = 0, onlyHighestDenomination = false) => {
  let distance = Math.abs(date1 - date2)

  if (!distance) {
    return '0s'
  }

  const days = Math.floor(distance / (24 * 3600000))
  if (onlyHighestDenomination && days) {
    return days + 'd'
  }
  distance -= days * 24 * 3600000

  const hours = Math.floor(distance / 3600000)
  if (onlyHighestDenomination && hours) {
    return hours + 'h'
  }
  distance -= hours * 3600000

  const minutes = Math.floor(distance / 60000)
  if (onlyHighestDenomination && minutes) {
    return minutes + 'm'
  }
  distance -= minutes * 60000

  const seconds = Math.floor(distance / 1000)
  if (onlyHighestDenomination) {
    return seconds + 's'
  }

  return `${days ? days + 'd ' : ''}${hours ? hours + 'h ' : ''}${
    minutes ? minutes + 'm' : hours || days ? '0m' : ''
  } ${seconds}s`
}

export const timeDifferenceInMinutesAndSeconds = (date1 = 0, date2 = 0) => {
  let distance = Math.abs(date1 - date2)

  if (!distance) {
    return '00:00'
  }

  const minutes = Math.floor(distance / 60000)
  distance -= minutes * 60000

  const seconds = Math.floor(distance / 1000)

  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
}

const LOCALE = Intl.NumberFormat().resolvedOptions?.().locale || 'en-US'

/**
 * Format a timestamp to short format time (i.e: 7:41 AM)
 * @param timestamp Timestamp
 * @param timeStyle Optional DateTimeFormat's `timeStyle` option.
 */
export function formatTime(timestamp: number | string, timeStyle = 'short'): string {
  return timestamp
    ? new Intl.DateTimeFormat(LOCALE, {
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore: TS built-in type for DateTimeFormat is not correct
        timeStyle
      }).format(new Date(timestamp))
    : ''
}

type FileDropCallback = (file: File) => void
//handle file drop in image upload
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const handleFileDrop = (event: any, callback: FileDropCallback): void => {
  event.preventDefault()

  const file = event?.dataTransfer?.files[0]
  if (file) {
    callback(file)
  }
}

type PasteCallback = (file: File) => void
// handle file paste in image upload
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const handlePaste = (event: { preventDefault: () => void; clipboardData: any }, callback: PasteCallback) => {
  event.preventDefault()
  const clipboardData = event.clipboardData
  const items = clipboardData.items

  if (items.length > 0) {
    const firstItem = items[0]
    if (firstItem.type.startsWith('image/') || firstItem.type.startsWith('video/')) {
      const blob = firstItem.getAsFile()
      callback(blob)
    }
  }
}

/**
 * Format a timestamp to medium format date (i.e: Jan 1, 2021)
 * @param timestamp Timestamp
 * @param dateStyle Optional DateTimeFormat's `dateStyle` option.
 */
export function formatDate(timestamp: number | string, dateStyle = 'medium'): string {
  return timestamp
    ? new Intl.DateTimeFormat(LOCALE, {
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore: TS built-in type for DateTimeFormat is not correct
        dateStyle
      }).format(new Date(timestamp))
    : ''
}

/**
 * Format a number with current locale.
 * @param num number
 * @returns Formatted string.
 */
export function formatNumber(num: number | bigint): string {
  return num ? new Intl.NumberFormat(LOCALE).format(num) : ''
}
export interface Violation {
  violation: string
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

export enum OrderSortDate {
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

export enum ApproveState {
  APPROVED = 'approved',
  CHANGEREQ = 'changereq',
  APPROVE = 'approve',
  OUTDATED = 'outdated'
}

export enum CheckStatus {
  PENDING = 'pending',
  RUNNING = 'running',
  SUCCESS = 'success',
  FAILURE = 'failure',
  ERROR = 'error',
  SKIPPED = 'skipped',
  KILLED = 'killed',
  FAILURE_IGNORED = 'failure_ignored'
}

export enum PRCommentFilterType {
  SHOW_EVERYTHING = 'showEverything',
  ALL_COMMENTS = 'allComments',
  MY_COMMENTS = 'myComments',
  RESOLVED_COMMENTS = 'resolvedComments',
  UNRESOLVED_COMMENTS = 'unresolvedComments'
}

const MONACO_SUPPORTED_LANGUAGES = [
  'abap',
  'apex',
  'azcli',
  'bat',
  'bicep',
  'cameligo',
  'clojure',
  'coffee',
  'cpp',
  'csharp',
  'csp',
  'css',
  'cypher',
  'dart',
  'dockerfile',
  'ecl',
  'elixir',
  'flow9',
  'freemarker2',
  'fsharp',
  'go',
  'graphql',
  'handlebars',
  'hcl',
  'html',
  'ini',
  'java',
  'javascript',
  'json',
  'julia',
  'kotlin',
  'less',
  'lexon',
  'liquid',
  'lua',
  'm3',
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
  'pla',
  'postiats',
  'powerquery',
  'powershell',
  'protobuf',
  'pug',
  'python',
  'qsharp',
  'r',
  'razor',
  'redis',
  'redshift',
  'restructuredtext',
  'ruby',
  'rust',
  'sb',
  'scala',
  'scheme',
  'scss',
  'shell',
  'solidity',
  'sophia',
  'sparql',
  'sql',
  'st',
  'swift',
  'systemverilog',
  'tcl',
  'twig',
  'typescript',
  'vb',
  'wgsl',
  'xml',
  'yaml'
]

// Some of the below languages are mapped to Monaco's built-in supported languages
// due to their similarity. We'll still need to get native support for them at
// some point.
const EXTENSION_TO_LANG: Record<string, string> = {
  alpine: 'dockerfile',
  bazel: 'python',
  cc: 'cpp',
  cs: 'csharp',
  env: 'shell',
  gitignore: 'shell',
  jsx: 'typescript',
  makefile: 'shell',
  toml: 'ini',
  tsx: 'typescript',
  tf: 'hcl',
  tfvars: 'hcl',
  workspace: 'python',
  tfstate: 'hcl',
  ipynb: 'json',
  mjs: 'javascript'
}

export const PLAIN_TEXT = 'plaintext'

export const filenameToLanguage = (name?: string): string | undefined => {
  const extension = (name?.split('.').pop() || '').toLowerCase()
  const lang = MONACO_SUPPORTED_LANGUAGES.find(l => l === extension) || EXTENSION_TO_LANG[extension]

  if (lang) {
    return lang
  }

  const map = langMap.languages(extension)

  if (map?.length) {
    return MONACO_SUPPORTED_LANGUAGES.find(_lang => map.includes(_lang)) || PLAIN_TEXT
  }

  return PLAIN_TEXT
}

export type EnumPublicKeyUsage = 'auth'

export interface TypeKeys {
  created: number
  verified?: number
  identifier: string
  usage: EnumPublicKeyUsage
  fingerprint: string
  comment: string
  type: string
}

interface WaitUtilParams<T> {
  test: () => T
  onMatched: (result: T) => void
  onExpired?: () => void
  duration?: number
  interval?: number
}

export interface inlineMergeFormValues {
  commitMessage: string
  commitTitle: string
}

export type inlineMergeFormRefType = FormikProps<inlineMergeFormValues>

export function waitUntil<T>({ test, onMatched, onExpired, duration = 5000, interval = 50 }: WaitUtilParams<T>) {
  const result = test()

  if (result) {
    onMatched(result)
  } else {
    if (duration > 0) {
      setTimeout(() => {
        waitUntil({ test, onMatched, onExpired, duration: duration - interval, interval })
      }, interval)
    } else {
      onExpired?.()
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

export enum CodeOwnerReqDecision {
  CHANGEREQ = 'changereq',
  APPROVED = 'approved',
  WAIT_FOR_APPROVAL = ''
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

export enum ScopeLevelEnum {
  ALL = 'all',
  CURRENT = 'current'
}

export enum PageAction {
  NEXT = 'next',
  PREV = 'previous'
}

export enum PullRequestCheckType {
  EMPTY = '',
  RAW = 'raw',
  MARKDOWN = 'markdown',
  PIPELINE = 'pipeline'
}

/**
 * Test if an element is close to viewport. Used to determine a progressive
 * pre-rendering before the element being scrolled to viewport.
 */
export function isInViewport(element: Element, margin = 0, direction: 'x' | 'y' | 'xy' = 'y') {
  const rect = element.getBoundingClientRect()

  const height = window.innerHeight || document.documentElement.clientHeight
  const width = window.innerWidth || document.documentElement.clientWidth
  const _top = 0 - margin
  const _bottom = height + margin
  const _left = 0 - margin
  const _right = width + margin

  const yCheck =
    (rect.top >= _top && rect.top <= _bottom) ||
    (rect.bottom >= _top && rect.bottom <= _bottom) ||
    (rect.top <= _top && rect.bottom >= _bottom)
  const xCheck = (rect.left >= _left && rect.left <= _right) || (rect.right >= _top && rect.right <= _right)

  if (direction === 'y') return yCheck
  if (direction === 'x') return xCheck

  return yCheck || xCheck
}

export const truncateString = (str: string, length: number): string =>
  str.length <= length ? str : str.slice(0, length - 3) + '...'

export const isIterable = (value: unknown): boolean => {
  // checks for null and undefined
  if (value == null) {
    return false
  }
  return Symbol.iterator in Object(value)
}

/**
 *
 * @param obj - Object to get all keys for (including nested fields)
 * @param prefix - Optional prefix for keys
 * @returns all keys with their prefixes
 */
export const getAllKeysWithPrefix = (obj: { [key: string]: string | boolean | object }, prefix = ''): string[] => {
  let keys: string[] = []
  for (const key in obj) {
    if (Object.prototype.hasOwnProperty.call(obj, key)) {
      const currentKey = prefix ? `${prefix}.${key}` : key
      keys.push(currentKey)
      if (typeof obj[key] === 'object') {
        // Recursively get keys from nested objects with updated prefix
        keys = keys.concat(getAllKeysWithPrefix(obj[key] as { [key: string]: string | boolean | object }, currentKey))
      }
    }
  }
  return keys
}

export const CustomEventName = {
  SIDE_NAV_EXPANDED_EVENT: 'SIDE_NAV_EXPANDED_EVENT'
} as const

export const PAGE_CONTAINER_WIDTH = '--page-container-width'

// Outlets are used to insert additional components into CommentBox
export enum CommentBoxOutletPosition {
  START_OF_MARKDOWN_EDITOR_TOOLBAR = 'start_of_markdown_editor_toolbar',
  ENABLE_AIDA_PR_DESC_BANNER = 'enable_aida_pr_desc_banner'
}

// Helper function to escape special characters for use in regex
export function escapeRegExp(str: string) {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') // $& means the whole matched string
}

export function removeSpecificTextOptimized(
  viewRef: React.MutableRefObject<EditorView | undefined>,
  textToRemove: string
) {
  const doc = viewRef?.current?.state.doc.toString()
  const regex = new RegExp(escapeRegExp(textToRemove), 'g')
  let match

  // Initialize an array to hold all changes
  const changes = []

  // Use regex to find all occurrences of the text
  while (doc && (match = regex.exec(doc)) !== null) {
    // Add a change object for each match to remove it
    changes.push({ from: match.index, to: match.index + match[0].length })
  }

  // Dispatch a single transaction with all changes if any matches were found
  if (changes.length > 0) {
    viewRef?.current?.dispatch({ changes })
  }
}

export const enum LabelType {
  DYNAMIC = 'dynamic',
  STATIC = 'static'
}

export enum ColorName {
  Red = 'red',
  Green = 'green',
  Yellow = 'yellow',
  Blue = 'blue',
  Pink = 'pink',
  Purple = 'purple',
  Violet = 'violet',
  Indigo = 'indigo',
  Cyan = 'cyan',
  Orange = 'orange',
  Brown = 'brown',
  Mint = 'mint',
  Lime = 'lime'
}

export interface ColorDetails {
  stroke: string
  background: string
  text: string
  backgroundWithoutStroke: string
}
export const colorsPanel: Record<ColorName, ColorDetails> = {
  [ColorName.Red]: { background: '#FFF7F7', stroke: '#F3A9AA', text: '#C7292F', backgroundWithoutStroke: '#FFE8EB' },
  [ColorName.Green]: { background: '#E9F9EE', stroke: '#85CBA2', text: '#16794C', backgroundWithoutStroke: '#E8F9ED' },
  [ColorName.Yellow]: { background: '#FFF9ED', stroke: '#E4B86F', text: '#92582D', backgroundWithoutStroke: '#FFF3D1' },
  [ColorName.Blue]: { background: '#F1FCFF', stroke: '#7BC8E0', text: '#236E93', backgroundWithoutStroke: '#E0F9FF' },
  [ColorName.Pink]: { background: '#FFF7FC', stroke: '#ECA8D2', text: '#C41B87', backgroundWithoutStroke: '#FEECF7' },
  [ColorName.Purple]: { background: '#FFF8FF', stroke: '#DFAAE3', text: '#9C2AAD', backgroundWithoutStroke: '#FCEDFC' },
  [ColorName.Violet]: { background: '#FBFAFF', stroke: '#C1B4F3', text: '#5645AF', backgroundWithoutStroke: '#F3F0FF' },
  [ColorName.Indigo]: { background: '#F8FAFF', stroke: '#A9BDF5', text: '#3250B2', backgroundWithoutStroke: '#EDF2FF' },
  [ColorName.Cyan]: { background: '#F2FCFD', stroke: '#7DCBD9', text: '#0B7792', backgroundWithoutStroke: '#E4F9FB' },
  [ColorName.Orange]: { background: '#FFF8F4', stroke: '#FFA778', text: '#995137', backgroundWithoutStroke: '#FFEDD5' },
  [ColorName.Brown]: { background: '#FCF9F6', stroke: '#DBB491', text: '#805C43', backgroundWithoutStroke: '#F8EFE7' },
  [ColorName.Mint]: { background: '#EFFEF9', stroke: '#7FD0BD', text: '#247469', backgroundWithoutStroke: '#DDFBF3' },
  [ColorName.Lime]: { background: '#F7FCF0', stroke: '#AFC978', text: '#586729', backgroundWithoutStroke: '#EDFADA' }
  // Add more colors when required
}

export const getColorsObj = (colorKey: ColorName): ColorDetails => {
  return colorsPanel[colorKey]
}

export function customEncodeURIComponent(str: string) {
  return encodeURIComponent(str).replace(/!/g, '%21')
}

export interface LabelTypes extends TypesLabel {
  labelValues?: TypesLabelValue[]
}

export enum LabelFilterType {
  LABEL = 'label',
  VALUE = 'value',
  FOR_VALUE = 'for_value'
}

export interface LabelFilterObj {
  labelId: number
  valueId?: number
  type: LabelFilterType
  labelObj: TypesLabel
  valueObj?: TypesLabelValue
}

export interface LabelListingProps {
  repoMetadata?: RepoRepositoryOutput
  space?: string
  activeTab?: string
}

export enum ScopeEnum {
  REPO_SCOPE = 0,
  ACCOUNT_SCOPE = 1,
  ORG_SCOPE = 2,
  PROJECT_SCOPE = 3,
  SPACE_SCOPE = 4
}

export interface ScopeData {
  scopeRef: string
  scopeIcon: IconName
  scopeId: string
  scopeName: ResourceType
  scopeColor: ColorName
}

export const getRelativeSpaceRef = (
  spaceScope: ScopeEnum,
  scope: number,
  orgIdentifier: string,
  projectIdentifier: string
): string | null =>
  spaceScope === ScopeEnum.ACCOUNT_SCOPE
    ? scope === ScopeEnum.PROJECT_SCOPE
      ? `${orgIdentifier}/${projectIdentifier}`
      : scope === ScopeEnum.ORG_SCOPE
      ? orgIdentifier
      : null
    : spaceScope === ScopeEnum.ORG_SCOPE && scope === ScopeEnum.PROJECT_SCOPE
    ? projectIdentifier
    : null

export const getScopeData = (space: string, scope: number, standalone: boolean): ScopeData => {
  const [accountId, orgIdentifier, projectIdentifier] = space.split('/')
  const scopeData: ScopeData = {
    scopeRef: space,
    scopeIcon: 'nav-project' as IconName,
    scopeId: space,
    scopeName: ResourceType.PROJECT,
    scopeColor: ColorName.Blue
  }

  if (standalone && scope !== ScopeEnum.REPO_SCOPE) {
    return { ...scopeData, scopeName: ResourceType.SPACE, scopeColor: ColorName.Indigo }
  }
  switch (scope) {
    case ScopeEnum.REPO_SCOPE:
      return {
        ...scopeData,
        scopeIcon: CodeIcon.Repo as IconName,
        scopeName: ResourceType.CODE_REPOSITORY,
        scopeColor: ColorName.Orange
      }
    case ScopeEnum.ACCOUNT_SCOPE:
      return {
        ...scopeData,
        scopeRef: accountId,
        scopeIcon: 'Account' as IconName,
        scopeId: accountId,
        scopeName: ResourceType.ACCOUNT,
        scopeColor: ColorName.Indigo
      }
    case ScopeEnum.ORG_SCOPE:
      return {
        ...scopeData,
        scopeRef: `${accountId}/${orgIdentifier}`,
        scopeIcon: 'nav-organization' as IconName,
        scopeId: orgIdentifier,
        scopeName: ResourceType.ORGANIZATION,
        scopeColor: ColorName.Purple
      }
    case ScopeEnum.PROJECT_SCOPE:
      return {
        ...scopeData,
        scopeRef: `${accountId}/${orgIdentifier}/${projectIdentifier}`,
        scopeId: projectIdentifier
      }
    default:
      return scopeData
  }
}

export const getScopeFromParams = (
  params: Identifier,
  standalone: boolean,
  repoMetadata?: RepoRepositoryOutput
): number => {
  if (repoMetadata) return ScopeEnum.REPO_SCOPE

  const { accountId, orgIdentifier, projectIdentifier } = params

  if (standalone) {
    return ScopeEnum.SPACE_SCOPE
  }

  if (accountId) {
    if (!orgIdentifier) {
      return ScopeEnum.ACCOUNT_SCOPE
    }

    return projectIdentifier ? ScopeEnum.PROJECT_SCOPE : ScopeEnum.ORG_SCOPE
  }

  return ScopeEnum.REPO_SCOPE
}

export const getScopeOptions = (
  getString: (key: StringKeys) => string,
  accountIdentifier: string,
  orgIdentifier?: string
): SelectOption[] =>
  [
    accountIdentifier && !orgIdentifier
      ? {
          label: getString('searchScope.allScopes'),
          value: ScopeLevelEnum.ALL
        }
      : null,
    accountIdentifier && !orgIdentifier
      ? { label: getString('searchScope.accOnly'), value: ScopeLevelEnum.CURRENT }
      : null,
    orgIdentifier ? { label: getString('searchScope.orgAndProj'), value: ScopeLevelEnum.ALL } : null,
    orgIdentifier ? { label: getString('searchScope.orgOnly'), value: ScopeLevelEnum.CURRENT } : null
  ].filter(Boolean) as SelectOption[]

export const getCurrentScopeLabel = (
  getString: (key: StringKeys) => string,
  scopeLevel: ScopeLevelEnum,
  accountIdentifier: string,
  orgIdentifier?: string
): SelectOption => {
  return scopeLevel === ScopeLevelEnum.ALL
    ? {
        label:
          accountIdentifier && !orgIdentifier
            ? getString('searchScope.allScopes')
            : getString('searchScope.orgAndProj'),
        value: ScopeLevelEnum.ALL
      }
    : {
        label:
          accountIdentifier && !orgIdentifier ? getString('searchScope.accOnly') : getString('searchScope.orgOnly'),
        value: ScopeLevelEnum.CURRENT
      }
}

export const getEditPermissionRequestFromScope = (
  space: string,
  scope: number,
  repoMetadata?: RepoRepositoryOutput
) => {
  const [accountIdentifier, orgIdentifier, projectIdentifier] = space.split('/')

  if (scope === ScopeEnum.REPO_SCOPE && repoMetadata) {
    return {
      resource: {
        resourceType: ResourceType.CODE_REPOSITORY,
        resourceIdentifier: repoMetadata?.identifier as string
      },
      permissions: [PermissionIdentifier.CODE_REPO_EDIT]
    }
  } else {
    switch (scope) {
      case 1:
        return {
          resource: {
            resourceType: ResourceType.ACCOUNT,
            resourceIdentifier: accountIdentifier as string
          },
          permissions: [PermissionIdentifier.UPDATE_ACCOUNT]
        }
      case 2:
        return {
          resource: {
            resourceType: ResourceType.ORGANIZATION,
            resourceIdentifier: orgIdentifier as string
          },
          permissions: [PermissionIdentifier.UPDATE_ORG]
        }
      case 3:
        return {
          resource: {
            resourceType: ResourceType.PROJECT,
            resourceIdentifier: projectIdentifier as string
          },
          permissions: [PermissionIdentifier.UPDATE_PROJECT]
        }
    }
  }
}

export const getEditPermissionRequestFromIdentifier = (space: string, repoMetadata?: RepoRepositoryOutput) => {
  const [accountIdentifier, orgIdentifier, projectIdentifier] = space.split('/')

  if (repoMetadata)
    return {
      resource: {
        resourceType: ResourceType.CODE_REPOSITORY,
        resourceIdentifier: repoMetadata?.identifier as string
      },
      permissions: [PermissionIdentifier.CODE_REPO_EDIT]
    }
  else if (projectIdentifier) {
    return {
      resource: {
        resourceType: ResourceType.PROJECT,
        resourceIdentifier: projectIdentifier as string
      },
      permissions: [PermissionIdentifier.UPDATE_PROJECT]
    }
  } else if (orgIdentifier) {
    return {
      resource: {
        resourceType: ResourceType.ORGANIZATION,
        resourceIdentifier: orgIdentifier as string
      },
      permissions: [PermissionIdentifier.UPDATE_ORG]
    }
  } else
    return {
      resource: {
        resourceType: ResourceType.ACCOUNT,
        resourceIdentifier: accountIdentifier as string
      },
      permissions: [PermissionIdentifier.UPDATE_ACCOUNT]
    }
}

export const replaceMentionIdWithEmail = (
  input: string,
  mentionsMap: {
    [key: string]: TypesPrincipalInfo
  }
) => input.replace(/@\[(\d+)\]/g, (match, id) => (mentionsMap[id] ? `@[${mentionsMap[id].email}]` : match))

export const replaceMentionEmailWithId = (
  input: string,
  emailMap: {
    [x: string]: TypesPrincipalInfo
  }
) => input.replace(/@\[(\S+@\S+\.\S+)\]/g, (match, email) => (emailMap[email] ? `@[${emailMap[email].id}]` : match))

export const roleStringKeyMap: Record<EnumMembershipRole, StringKeys> = {
  contributor: 'contributor',
  executor: 'executor',
  reader: 'reader',
  space_owner: 'owner'
}

export const formatListWithAnd = (list: string[]): string => {
  if (list) {
    if (list.length === 0) return ''
    if (list.length === 1) return list[0]
    if (list.length === 2) return list.join(' and ')
    return `${list.slice(0, -1).join(', ')} and ${list[list.length - 1]}`
  } else return ''
}

/**
 * Normalizes and combines principal and user group data into a unified format
 * @param principals - Array of principal objects from principals API
 * @param userGroups - Array of user group objects from usergroups API
 * @returns Combined array of normalized objects with consistent structure
 */
export interface NormalizedPrincipal {
  id: number
  email_or_identifier: string
  type: PrincipalType
  display_name: string
}

export function combineAndNormalizePrincipalsAndGroups(
  principals: TypesPrincipalInfo[] | null,
  userGroups?: TypesUserGroupInfo[] | null,
  notSorted?: boolean
): NormalizedPrincipal[] {
  const normalizedData: NormalizedPrincipal[] = []

  // Process user groups data if available
  if (userGroups && Array.isArray(userGroups)) {
    userGroups.forEach(group => {
      normalizedData.push({
        id: group.id || -1,
        email_or_identifier: group.identifier || '',
        type: PrincipalType.USER_GROUP,
        display_name: group.name || group.identifier || 'Unknown Group'
      })
    })
  }

  // Process principals data if available
  if (principals && Array.isArray(principals)) {
    principals.forEach(principal => {
      normalizedData.push({
        id: principal.id || -1,
        email_or_identifier: principal.email || principal.uid || '',
        type: (principal.type as PrincipalType) || PrincipalType.USER,
        display_name: principal.display_name || principal.email || 'Unknown User'
      })
    })
  }

  if (notSorted) return normalizedData

  return normalizedData.sort((a, b) => a.display_name.localeCompare(b.display_name))
}

export interface TypesPrincipalInfoWithReviewDecision extends TypesPrincipalInfo {
  review_decision?: PullReqReviewDecision
  review_sha?: string
}

export interface TypesDefaultReviewerApprovalsResponseWithRevDecision extends TypesDefaultReviewerApprovalsResponse {
  principals?: TypesPrincipalInfoWithReviewDecision[] | null // Override the 'principals' field
}
export const getUnifiedDefaultReviewersState = (info: TypesDefaultReviewerApprovalsResponseWithRevDecision[]) => {
  const defaultReviewState = {
    defReviewerApprovalRequiredByRule: false,
    defReviewerLatestApprovalRequiredByRule: false,
    defReviewerApprovedLatestChanges: true,
    defReviewerApprovedChanges: true,
    changesRequestedByDefReviewersArr: [] as TypesPrincipalInfoWithReviewDecision[]
  }

  info?.forEach(item => {
    if (item?.minimum_required_count !== undefined && item.minimum_required_count > 0) {
      defaultReviewState.defReviewerApprovalRequiredByRule = true
      if (item.current_count !== undefined && item.current_count < item.minimum_required_count) {
        defaultReviewState.defReviewerApprovedChanges = false
      }
    }
    if (item?.minimum_required_count_latest !== undefined && item.minimum_required_count_latest > 0) {
      defaultReviewState.defReviewerLatestApprovalRequiredByRule = true
      if (item.current_count !== undefined && item.current_count < item.minimum_required_count_latest) {
        defaultReviewState.defReviewerApprovedLatestChanges = false
      }
    }

    item?.principals?.forEach(principal => {
      if (principal?.review_decision === PullReqReviewDecision.CHANGEREQ)
        defaultReviewState.changesRequestedByDefReviewersArr.push(principal)
    })
  })

  return defaultReviewState
}
