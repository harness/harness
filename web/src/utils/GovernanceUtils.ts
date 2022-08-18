import { Intent, IToaster, IToastProps, Position, Toaster } from '@blueprintjs/core'
import type { editor as EDITOR } from 'monaco-editor/esm/vs/editor/editor.api'
import { Color } from '@harness/uicore'
import { get } from 'lodash-es'
import moment from 'moment'
import { useParams } from 'react-router-dom'
import { useEffect } from 'react'
import type { StringsContextValue } from 'framework/strings/StringsContext'
import { useAppContext } from 'AppContext'
import { useStandaloneFeatureFlags } from '../hooks/useStandaloneFeatureFlags'

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
  tabSize: 4,
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

export const ENTITIES = {
  pipeline: {
    label: 'Pipeline',
    value: 'pipeline',
    eventTypes: [
      {
        label: 'Pipeline Evaluation',
        value: 'evaluation'
      }
    ],
    actions: [
      { label: 'On Run', value: 'onrun' },
      { label: 'On Save', value: 'onsave' }
      // {
      //   label: 'On Step',
      //   value: 'onstep',
      //   enableAction: flags => {
      //     return flags?.CUSTOM_POLICY_STEP
      //   }
      // }
    ],
    enabledFunc: flags => {
      return flags?.OPA_PIPELINE_GOVERNANCE
    }
  },
  flag: {
    label: 'Feature Flag',
    value: 'flag',
    eventTypes: [
      {
        label: 'Flag Evaluation',
        value: 'flag_evaluation'
      }
    ],
    actions: [{ label: 'On Save', value: 'onsave' }],
    enabledFunc: flags => {
      return flags?.OPA_FF_GOVERNANCE
    }
  },
  connector: {
    label: 'Connector',
    value: 'connector',
    eventTypes: [
      {
        label: 'Connector Evaluation',
        value: 'connector_evaluation'
      }
    ],
    actions: [{ label: 'On Save', value: 'onsave' }],
    enabledFunc: flags => {
      return flags?.OPA_CONNECTOR_GOVERNANCE
    }
  },
  secret: {
    label: 'Secret',
    value: 'secret',
    eventTypes: [
      {
        label: 'On Save',
        value: 'onsave'
      }
    ],
    actions: [{ label: 'On Save', value: 'onsave' }],
    enabledFunc: flags => {
      return flags?.OPA_SECRET_GOVERNANCE
    }
  },
  custom: {
    label: 'Custom',
    value: 'custom',
    eventTypes: [
      {
        label: 'Custom Evaluation',
        value: 'custom_evaluation'
      }
    ],
    actions: [{ label: 'On Step', value: 'onstep' }],
    enabledFunc: flags => {
      return flags?.CUSTOM_POLICY_STEP
    }
  }
} as Entities

export const getEntityLabel = (entity: keyof Entities): string => {
  return ENTITIES[entity].label
}

export function useEntities(): Entities {
  const {
    hooks: { useFeatureFlags = useStandaloneFeatureFlags }
  } = useAppContext()
  const flags = useFeatureFlags()
  const availableEntities = { ...ENTITIES }

  for (const key in ENTITIES) {
    if (!ENTITIES[key as keyof Entities].enabledFunc(flags)) {
      delete availableEntities[key as keyof Entities]
      continue
    }

    // temporary(?) feature flagging of actions
    availableEntities[key as keyof Entities].actions = availableEntities[key as keyof Entities].actions.filter(
      action => {
        return action.enableAction ? action.enableAction(flags) : true
      }
    )
  }
  return availableEntities
}

export const getActionType = (type: string | undefined, action: string | undefined): string => {
  return ENTITIES[type as keyof Entities].actions.find(a => a.value === action)?.label || 'Unrecognised Action Type'
}

export type FeatureFlagMap = Partial<Record<FeatureFlag, boolean>>

export enum FeatureFlag {
  OPA_PIPELINE_GOVERNANCE = 'OPA_PIPELINE_GOVERNANCE',
  OPA_FF_GOVERNANCE = 'OPA_FF_GOVERNANCE',
  CUSTOM_POLICY_STEP = 'CUSTOM_POLICY_STEP',
  OPA_CONNECTOR_GOVERNANCE = 'OPA_CONNECTOR_GOVERNANCE',
  OPA_GIT_GOVERNANCE = 'OPA_GIT_GOVERNANCE',
  OPA_SECRET_GOVERNANCE = 'OPA_SECRET_GOVERNANCE'
}

export type Entity = {
  label: string
  value: string
  eventTypes: Event[]
  actions: Action[]
  enabledFunc: (flags: FeatureFlagMap) => boolean
}

export type Event = {
  label: string
  value: string
}

export type Action = {
  label: string
  value: string
  enableAction?: (flags: FeatureFlagMap) => boolean
}

export type Entities = {
  pipeline: Entity
  flag: Entity
  connector: Entity
  secret: Entity
  custom: Entity
}

export enum EvaluationStatus {
  ERROR = 'error',
  PASS = 'pass',
  WARNING = 'warning'
}

export const isEvaluationFailed = (status?: string): boolean =>
  status === EvaluationStatus.ERROR || status === EvaluationStatus.WARNING

export const LIST_FETCHING_PAGE_SIZE = 20

// TODO - we should try and drive all these from the ENTITIES const ^ as well
// theres still a little duplication going on
export enum PipeLineEvaluationEvent {
  ON_RUN = 'onrun',
  ON_SAVE = 'onsave',
  ON_CREATE = 'oncreate',
  ON_STEP = 'onstep'
}

// TODO - we should try and drive all these from the ENTITIES const ^ as well
// theres still a little duplication going on
export enum PolicySetType {
  PIPELINE = 'pipeline',
  FEATURE_FLAGS = 'flag',
  CUSTOM = 'custom',
  CONNECTOR = 'connector'
}

export const getEvaluationEventString = (
  getString: StringsContextValue['getString'],
  evaluation: PipeLineEvaluationEvent
): string => {
  if (!getString) return ''

  const evaluations = {
    onrun: getString('governance.onRun'),
    onsave: getString('governance.onSave'),
    oncreate: getString('governance.onCreate'),
    onstep: getString('governance.onStep')
  }

  return evaluations[evaluation]
}

export const getEvaluationNameString = (evaluationMetadata: string): string | undefined => {
  try {
    const entityMetadata = JSON.parse(decodeURIComponent(evaluationMetadata as string))
    if (entityMetadata.entityName) {
      return entityMetadata.entityName
    } else if (entityMetadata['pipelineName']) {
      return entityMetadata['pipelineName'] //temporary until pipelineName is not being used
    } else {
      return 'Unknown'
    }
  } catch {
    return 'Unknown'
  }
}

export const evaluationStatusToColor = (status: string): Color => {
  switch (status) {
    case EvaluationStatus.ERROR:
      return Color.ERROR
    case EvaluationStatus.WARNING:
      return Color.WARNING
  }

  return Color.SUCCESS
}

// @see https://github.com/drone/policy-mgmt/issues/270
// export const QUERY_PARAM_VALUE_ALL = '*'

export const DEFAULT_DATE_FORMAT = 'MM/DD/YYYY hh:mm a'

interface SetPageNumberProps {
  setPage: (value: React.SetStateAction<number>) => void
  pageItemsCount?: number
  page: number
}

export const setPageNumber = ({ setPage, pageItemsCount, page }: SetPageNumberProps): void => {
  if (pageItemsCount === 0 && page > 0) {
    setPage(page - 1)
  }
}

export const ILLEGAL_IDENTIFIERS = [
  'or',
  'and',
  'eq',
  'ne',
  'lt',
  'gt',
  'le',
  'ge',
  'div',
  'mod',
  'not',
  'null',
  'true',
  'false',
  'new',
  'var',
  'return'
]

export const REGO_MONACO_LANGUAGE_IDENTIFIER = 'rego'

export const omit = (originalObj = {}, keysToOmit: string[]) =>
  Object.fromEntries(Object.entries(originalObj).filter(([key]) => !keysToOmit.includes(key)))

export const displayDateTime = (value: number): string | null => {
  return value ? moment.unix(value / 1000).format(DEFAULT_DATE_FORMAT) : null
}

export interface GitFilterScope {
  repo?: string
  branch?: GitBranchDTO['branchName']
  getDefaultFromOtherRepo?: boolean
}

export interface GitFiltersProps {
  defaultValue?: GitFilterScope
  onChange: (value: GitFilterScope) => void
  className?: string
  branchSelectClassName?: string
  showRepoSelector?: boolean
  showBranchSelector?: boolean
  showBranchIcon?: boolean
  shouldAllowBranchSync?: boolean
  getDisabledOptionTitleText?: () => string
}

export interface GitBranchDTO {
  branchName?: string
  branchSyncStatus?: 'SYNCED' | 'SYNCING' | 'UNSYNCED'
}

type Module = 'cd' | 'cf' | 'ci' | undefined

export const useGetModuleQueryParam = (): Module => {
  const { projectIdentifier, module } = useParams<Record<string, string>>()
  return projectIdentifier ? (module as Module) : undefined
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

export const useGetTrialInfo = (): any => {
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
