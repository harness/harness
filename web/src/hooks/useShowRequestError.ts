import { useToaster } from '@harness/uicore'
import { useEffect } from 'react'
import type { GetDataError } from 'restful-react'
import { getErrorMessage } from 'utils/Utils'

export function useShowRequestError(error: GetDataError<Unknown> | null) {
  const { showError } = useToaster()

  useEffect(() => {
    if (error) {
      showError(getErrorMessage(error))
    }
  }, [error, showError])
}
