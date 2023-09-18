import { useToaster } from '@harnessio/uicore'
import { useEffect } from 'react'
import type { GetDataError } from 'restful-react'
import { getErrorMessage } from 'utils/Utils'

export function useShowRequestError(error: GetDataError<Unknown> | null, timeout?: number) {
  const { showError } = useToaster()

  useEffect(() => {
    if (error) {
      showError(getErrorMessage(error), timeout)
    }
  }, [error, showError, timeout])
}
