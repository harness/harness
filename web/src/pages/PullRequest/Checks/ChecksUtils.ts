import { ExecutionState } from 'components/ExecutionStatus/ExecutionStatus'
import type { EnumCheckStatus } from 'services/code'

type CheckType = { status: EnumCheckStatus }[]

export function findDefaultExecution<T>(collection: Iterable<T> | null | undefined) {
  return (collection as CheckType)?.length
    ? (((collection as CheckType).find(({ status }) => status === ExecutionState.ERROR) ||
        (collection as CheckType).find(({ status }) => status === ExecutionState.FAILURE) ||
        (collection as CheckType).find(({ status }) => status === ExecutionState.RUNNING) ||
        (collection as CheckType).find(({ status }) => status === ExecutionState.SUCCESS) ||
        (collection as CheckType).find(({ status }) => status === ExecutionState.PENDING) ||
        (collection as CheckType)[0]) as T)
    : null
}
