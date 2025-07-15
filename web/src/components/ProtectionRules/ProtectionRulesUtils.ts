import type { UseStringsReturn } from 'framework/strings'
import type { EnumMergeMethod, OpenapiRule, OpenapiRuleType, ProtectionBranch, TypesPrincipalInfo } from 'services/code'
import { MergeStrategy, ProtectionRulesType, RulesTargetType } from 'utils/GitUtils'
import { PrincipalType } from 'utils/Utils'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const transformDataToArray = (data: any) => {
  return Object.keys(data).map(key => {
    return {
      ...data[key]
    }
  })
}

export enum RuleState {
  ACTIVE = 'active',
  MONITOR = 'monitor',
  DISABLED = 'disabled'
}

export type RuleFieldsMap = Record<RuleFields, boolean>

export type Rule = {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  [key: string]: any
}

export type ProtectionRule = {
  title: string
  requiredRule: {
    [key in RuleFields]?: boolean
  }
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
  userGroups?: any[]
): NormalizedPrincipal[] {
  const normalizedData: NormalizedPrincipal[] = []

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

  // Process user groups data if available
  if (userGroups && Array.isArray(userGroups)) {
    userGroups.forEach(group => {
      normalizedData.push({
        id: group.id || '',
        email_or_identifier: group.identifier || '',
        type: PrincipalType.USER_GROUP,
        display_name: group.name || group.identifier || 'Unknown Group'
      })
    })
  }

  return normalizedData.sort((a, b) => a.display_name.localeCompare(b.display_name))
}

export type RulesFormPayload = {
  name?: string
  desc?: string
  enable: boolean
  target?: string
  targetDefault?: boolean
  targetList: string[][]
  allRepoOwners?: boolean
  bypassList?: NormalizedPrincipal[]
  defaultReviewersList?: string[]
  requireMinReviewers: boolean
  requireMinDefaultReviewers: boolean
  minReviewers?: string | number
  minDefaultReviewers?: string | number
  autoAddCodeOwner?: boolean
  requireCodeOwner?: boolean
  requireNewChanges?: boolean
  reqResOfChanges?: boolean
  requireCommentResolution?: boolean
  requireStatusChecks: boolean
  statusChecks: string[]
  limitMergeStrategies: boolean
  mergeCommit?: boolean
  squashMerge?: boolean
  rebaseMerge?: boolean
  fastForwardMerge?: boolean
  autoDelete?: boolean
  blockCreation?: boolean
  blockDeletion?: boolean
  blockUpdate?: boolean
  blockForcePush?: boolean
  requirePr?: boolean
  bypassSet: boolean
  targetSet: boolean
  defaultReviewersSet: boolean
  defaultReviewersEnabled: boolean
}

export enum RuleFields {
  APPROVALS_REQUIRE_MINIMUM_COUNT = 'pullreq.approvals.require_minimum_count',
  APPROVALS_REQUIRE_CODE_OWNERS = 'pullreq.approvals.require_code_owners',
  APPROVALS_REQUIRE_NO_CHANGE_REQUEST = 'pullreq.approvals.require_no_change_request',
  APPROVALS_REQUIRE_MINIMUM_DEFAULT_REVIEWERS = 'pullreq.approvals.require_minimum_default_reviewer_count',
  APPROVALS_REQUIRE_LATEST_COMMIT = 'pullreq.approvals.require_latest_commit',
  AUTO_ADD_CODE_OWNERS = 'pullreq.reviewers.request_code_owners',
  DEFAULT_REVIEWERS_ADDED = 'pullreq.reviewers.default_reviewer_ids',
  COMMENTS_REQUIRE_RESOLVE_ALL = 'pullreq.comments.require_resolve_all',
  STATUS_CHECKS_ALL_MUST_SUCCEED = 'pullreq.status_checks.all_must_succeed',
  STATUS_CHECKS_REQUIRE_IDENTIFIERS = 'pullreq.status_checks.require_identifiers',
  MERGE_STRATEGIES_ALLOWED = 'pullreq.merge.strategies_allowed',
  MERGE_DELETE_BRANCH = 'pullreq.merge.delete_branch',
  LIFECYCLE_CREATE_FORBIDDEN = 'lifecycle.create_forbidden',
  LIFECYCLE_DELETE_FORBIDDEN = 'lifecycle.delete_forbidden',
  MERGE_BLOCK = 'pullreq.merge.block',
  LIFECYCLE_UPDATE_FORBIDDEN = 'lifecycle.update_forbidden',
  LIFECYCLE_UPDATE_FORCE_FORBIDDEN = 'lifecycle.update_force_forbidden'
}

export type ProtectionRulesMapType = Record<string, ProtectionRule>

export function createRuleFieldsMap(ruleDefinition: Rule): RuleFieldsMap {
  const ruleFieldsMap: RuleFieldsMap = {
    [RuleFields.APPROVALS_REQUIRE_MINIMUM_COUNT]: false,
    [RuleFields.APPROVALS_REQUIRE_CODE_OWNERS]: false,
    [RuleFields.APPROVALS_REQUIRE_NO_CHANGE_REQUEST]: false,
    [RuleFields.APPROVALS_REQUIRE_LATEST_COMMIT]: false,
    [RuleFields.APPROVALS_REQUIRE_MINIMUM_DEFAULT_REVIEWERS]: false,
    [RuleFields.AUTO_ADD_CODE_OWNERS]: false,
    [RuleFields.DEFAULT_REVIEWERS_ADDED]: false,
    [RuleFields.COMMENTS_REQUIRE_RESOLVE_ALL]: false,
    [RuleFields.STATUS_CHECKS_ALL_MUST_SUCCEED]: false,
    [RuleFields.STATUS_CHECKS_REQUIRE_IDENTIFIERS]: false,
    [RuleFields.MERGE_STRATEGIES_ALLOWED]: false,
    [RuleFields.MERGE_DELETE_BRANCH]: false,
    [RuleFields.LIFECYCLE_CREATE_FORBIDDEN]: false,
    [RuleFields.LIFECYCLE_DELETE_FORBIDDEN]: false,
    [RuleFields.MERGE_BLOCK]: false,
    [RuleFields.LIFECYCLE_UPDATE_FORBIDDEN]: false,
    [RuleFields.LIFECYCLE_UPDATE_FORCE_FORBIDDEN]: false
  }
  if (ruleDefinition?.pullreq) {
    if (ruleDefinition.pullreq.approvals) {
      ruleFieldsMap[RuleFields.APPROVALS_REQUIRE_CODE_OWNERS] = !!ruleDefinition.pullreq.approvals.require_code_owners
      ruleFieldsMap[RuleFields.APPROVALS_REQUIRE_LATEST_COMMIT] =
        !!ruleDefinition.pullreq.approvals.require_latest_commit
      ruleFieldsMap[RuleFields.APPROVALS_REQUIRE_MINIMUM_COUNT] =
        typeof ruleDefinition.pullreq.approvals.require_minimum_count === 'number'
      ruleFieldsMap[RuleFields.APPROVALS_REQUIRE_NO_CHANGE_REQUEST] =
        !!ruleDefinition.pullreq.approvals.require_no_change_request
      ruleFieldsMap[RuleFields.APPROVALS_REQUIRE_MINIMUM_DEFAULT_REVIEWERS] =
        !!ruleDefinition.pullreq.approvals.require_minimum_default_reviewer_count
    }

    if (ruleDefinition.pullreq.comments) {
      ruleFieldsMap[RuleFields.COMMENTS_REQUIRE_RESOLVE_ALL] = !!ruleDefinition.pullreq.comments.require_resolve_all
    }

    if (ruleDefinition.pullreq.merge) {
      ruleFieldsMap[RuleFields.MERGE_BLOCK] = !!ruleDefinition.pullreq.merge.block
      ruleFieldsMap[RuleFields.MERGE_DELETE_BRANCH] = !!ruleDefinition.pullreq.merge.delete_branch
      ruleFieldsMap[RuleFields.MERGE_STRATEGIES_ALLOWED] =
        Array.isArray(ruleDefinition.pullreq.merge.strategies_allowed) &&
        ruleDefinition.pullreq.merge.strategies_allowed.length > 0
    }

    if (ruleDefinition.pullreq.status_checks) {
      ruleFieldsMap[RuleFields.STATUS_CHECKS_REQUIRE_IDENTIFIERS] =
        Array.isArray(ruleDefinition.pullreq.status_checks.require_identifiers) &&
        ruleDefinition.pullreq.status_checks.require_identifiers.length > 0
    }

    if (ruleDefinition.pullreq.reviewers) {
      ruleFieldsMap[RuleFields.AUTO_ADD_CODE_OWNERS] = !!ruleDefinition.pullreq.reviewers.request_code_owners
      ruleFieldsMap[RuleFields.DEFAULT_REVIEWERS_ADDED] =
        Array.isArray(ruleDefinition.pullreq.reviewers.default_reviewer_ids) &&
        ruleDefinition.pullreq.reviewers.default_reviewer_ids.length > 0
    }
  }

  if (ruleDefinition?.lifecycle) {
    ruleFieldsMap[RuleFields.LIFECYCLE_CREATE_FORBIDDEN] = !!ruleDefinition.lifecycle.create_forbidden
    ruleFieldsMap[RuleFields.LIFECYCLE_DELETE_FORBIDDEN] = !!ruleDefinition.lifecycle.delete_forbidden
    ruleFieldsMap[RuleFields.LIFECYCLE_UPDATE_FORBIDDEN] = !!ruleDefinition.lifecycle.update_forbidden
    ruleFieldsMap[RuleFields.LIFECYCLE_UPDATE_FORCE_FORBIDDEN] = !!ruleDefinition.lifecycle.update_force_forbidden
  }

  return ruleFieldsMap
}

export const getProtectionRules = (getString: UseStringsReturn['getString'], ruleType?: OpenapiRuleType) => {
  const rules = {
    blockCreation: {
      title: getString('protectionRules.blockCreation', { ruleType }),
      requiredRule: {
        [RuleFields.LIFECYCLE_CREATE_FORBIDDEN]: true
      }
    },
    blockDeletion: {
      title: getString('protectionRules.blockDeletion', { ruleType }),
      requiredRule: {
        [RuleFields.LIFECYCLE_DELETE_FORBIDDEN]: true
      }
    },
    blockUpdate: {
      title: getString('protectionRules.blockUpdate', { ruleType }),
      requiredRule: {
        [RuleFields.LIFECYCLE_UPDATE_FORCE_FORBIDDEN]: true
      }
    }
  }

  switch (ruleType) {
    case ProtectionRulesType.BRANCH:
      return {
        ...rules,
        blockUpdate: {
          title: getString('protectionRules.blockUpdate', { ruleType }),
          requiredRule: {
            [RuleFields.MERGE_BLOCK]: true,
            [RuleFields.LIFECYCLE_UPDATE_FORBIDDEN]: true
          }
        },
        requireMinReviewersTitle: {
          title: getString('protectionRules.requireMinReviewersTitle'),
          requiredRule: {
            [RuleFields.APPROVALS_REQUIRE_MINIMUM_COUNT]: true
          }
        },
        reqReviewFromCodeOwnerTitle: {
          title: getString('protectionRules.reqReviewFromCodeOwnerTitle'),
          requiredRule: {
            [RuleFields.APPROVALS_REQUIRE_CODE_OWNERS]: true
          }
        },
        reqResOfChanges: {
          title: getString('protectionRules.reqResOfChanges'),
          requiredRule: {
            [RuleFields.APPROVALS_REQUIRE_NO_CHANGE_REQUEST]: true
          }
        },
        reqNewChangesTitle: {
          title: getString('protectionRules.reqNewChangesTitle'),
          requiredRule: {
            [RuleFields.APPROVALS_REQUIRE_LATEST_COMMIT]: true
          }
        },
        reqCommentResolutionTitle: {
          title: getString('protectionRules.reqCommentResolutionTitle'),
          requiredRule: {
            [RuleFields.COMMENTS_REQUIRE_RESOLVE_ALL]: true
          }
        },
        reqStatusChecksTitleAll: {
          title: getString('protectionRules.reqStatusChecksTitle'),
          requiredRule: {
            [RuleFields.STATUS_CHECKS_ALL_MUST_SUCCEED]: true
          }
        },
        reqStatusChecksTitle: {
          title: getString('protectionRules.reqStatusChecksTitle'),
          requiredRule: {
            [RuleFields.STATUS_CHECKS_REQUIRE_IDENTIFIERS]: true
          }
        },
        limitMergeStrategies: {
          title: getString('protectionRules.limitMergeStrategies'),
          requiredRule: {
            [RuleFields.MERGE_STRATEGIES_ALLOWED]: true
          }
        },
        autoDeleteTitle: {
          title: getString('protectionRules.autoDeleteTitle'),
          requiredRule: {
            [RuleFields.MERGE_DELETE_BRANCH]: true
          }
        },
        requirePr: {
          title: getString('protectionRules.requirePr'),
          requiredRule: {
            [RuleFields.LIFECYCLE_UPDATE_FORBIDDEN]: true,
            [RuleFields.MERGE_BLOCK]: false
          }
        },
        blockForcePush: {
          title: getString('protectionRules.blockForcePush'),
          requiredRule: {
            [RuleFields.LIFECYCLE_UPDATE_FORCE_FORBIDDEN]: true
          }
        },
        autoAddCodeownersToReview: {
          title: getString('protectionRules.addCodeownersToReviewTitle'),
          requiredRule: {
            [RuleFields.AUTO_ADD_CODE_OWNERS]: true
          }
        },
        requireMinDefaultReviewersTitle: {
          title: getString('protectionRules.requireMinDefaultReviewersTitle'),
          requiredRule: {
            [RuleFields.APPROVALS_REQUIRE_MINIMUM_DEFAULT_REVIEWERS]: true
          }
        },
        defaultReviewersAdded: {
          title: getString('protectionRules.enableDefaultReviewersTitle'),
          requiredRule: {
            [RuleFields.DEFAULT_REVIEWERS_ADDED]: true
          }
        }
      }
    case ProtectionRulesType.TAG:
      return rules
  }

  return rules
}

export const rulesFormInitialPayload: RulesFormPayload = {
  name: '',
  desc: '',
  enable: true,
  target: '',
  targetDefault: false,
  targetList: [] as string[][],
  allRepoOwners: false,
  bypassList: [] as NormalizedPrincipal[],
  defaultReviewersList: [] as string[],
  requireMinReviewers: false,
  requireMinDefaultReviewers: false,
  minReviewers: '',
  minDefaultReviewers: '',
  requireCodeOwner: false,
  requireNewChanges: false,
  reqResOfChanges: false,
  requireCommentResolution: false,
  requireStatusChecks: false,
  statusChecks: [] as string[],
  limitMergeStrategies: false,
  mergeCommit: false,
  squashMerge: false,
  rebaseMerge: false,
  autoDelete: false,
  blockCreation: false,
  blockDeletion: false,
  blockUpdate: false,
  blockForcePush: false,
  requirePr: false,
  bypassSet: false,
  targetSet: false,
  defaultReviewersSet: false,
  defaultReviewersEnabled: false
}

export const getPayload = (formData: RulesFormPayload, ruleType: OpenapiRuleType): OpenapiRule => {
  const stratArray = [
    formData.squashMerge && MergeStrategy.SQUASH,
    formData.rebaseMerge && MergeStrategy.REBASE,
    formData.mergeCommit && MergeStrategy.MERGE,
    formData.fastForwardMerge && MergeStrategy.FAST_FORWARD
  ].filter(Boolean) as EnumMergeMethod[]
  const includeArray =
    formData?.targetList?.filter(([type]) => type === RulesTargetType.INCLUDE).map(([, value]) => value) ?? []
  const excludeArray =
    formData?.targetList?.filter(([type]) => type === RulesTargetType.EXCLUDE).map(([, value]) => value) ?? []

  const { userIds, userGroupIds } = formData?.bypassList?.reduce(
    (acc, item: NormalizedPrincipal) => {
      if (item.type === PrincipalType.USER_GROUP) {
        acc.userGroupIds.push(item.id)
      } else {
        acc.userIds.push(item.id)
      }
      return acc
    },
    { userIds: [] as number[], userGroupIds: [] as number[] }
  ) || { userIds: [], userGroupIds: [] }

  const isBranchRuleType = ruleType === ProtectionRulesType.BRANCH

  const payload = {
    identifier: formData.name,
    type: ruleType,
    description: formData.desc,
    state: formData.enable ? RuleState.ACTIVE : RuleState.DISABLED,
    pattern: {
      default: formData.targetDefault,
      exclude: excludeArray,
      include: includeArray
    },
    definition: {
      bypass: {
        user_ids: userIds,
        user_group_ids: userGroupIds,
        repo_owners: formData.allRepoOwners
      },
      ...(isBranchRuleType
        ? {
            pullreq: {
              approvals: {
                require_code_owners: formData.requireCodeOwner,
                require_minimum_count: parseInt(formData.minReviewers as string),
                require_minimum_default_reviewer_count: parseInt(formData.minDefaultReviewers as string),
                require_latest_commit: formData.requireNewChanges,
                require_no_change_request: formData.reqResOfChanges
              },
              reviewers: {
                request_code_owners: formData.autoAddCodeOwner,
                default_reviewer_ids: formData?.defaultReviewersList?.map(item => parseInt(item.split(' ')[0]))
              },
              comments: {
                require_resolve_all: formData.requireCommentResolution
              },
              merge: {
                strategies_allowed: stratArray,
                delete_branch: formData.autoDelete,
                block: formData.blockUpdate
              },
              status_checks: {
                require_identifiers: formData.statusChecks
              }
            }
          }
        : {}),
      lifecycle: {
        create_forbidden: formData.blockCreation,
        delete_forbidden: formData.blockDeletion,
        ...(isBranchRuleType
          ? {
              update_forbidden: formData.requirePr || formData.blockUpdate,
              update_force_forbidden: formData.blockForcePush && !formData.requirePr && !formData.blockUpdate
            }
          : { update_force_forbidden: formData.blockUpdate })
      }
    }
  }
  if (!formData.requireStatusChecks) {
    delete (payload?.definition as ProtectionBranch)?.pullreq?.status_checks
  }
  if (!formData.limitMergeStrategies) {
    delete (payload?.definition as ProtectionBranch)?.pullreq?.merge?.strategies_allowed
  }
  if (!formData.requireMinReviewers) {
    delete (payload?.definition as ProtectionBranch)?.pullreq?.approvals?.require_minimum_count
  }
  if (!formData.requireMinDefaultReviewers) {
    delete (payload?.definition as ProtectionBranch)?.pullreq?.approvals?.require_minimum_default_reviewer_count
  }

  return payload
}
