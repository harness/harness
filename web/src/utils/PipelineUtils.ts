import { ExecutionState } from 'components/ExecutionStatus/ExecutionStatus'

export const getStatus = (status: string | undefined): ExecutionState => {
  switch (status) {
    case 'success':
      return ExecutionState.SUCCESS
    case 'failed':
      return ExecutionState.FAILURE
    case 'running':
      return ExecutionState.RUNNING
    case 'pending':
      return ExecutionState.PENDING
    case 'error':
      return ExecutionState.ERROR
    default:
      return ExecutionState.PENDING
  }
}
