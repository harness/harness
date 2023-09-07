import { ExecutionState } from 'components/ExecutionStatus/ExecutionStatus'

export const getStatus = (status: string): ExecutionState => {
  switch (status) {
    case 'success':
      return ExecutionState.SUCCESS
    case 'failed':
      return ExecutionState.FAILURE
    case 'running':
      return ExecutionState.RUNNING
    case 'pending':
      return ExecutionState.PENDING
    default:
      return ExecutionState.PENDING
  }
}
