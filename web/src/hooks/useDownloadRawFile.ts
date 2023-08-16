import { useCallback, useEffect, useState } from 'react'
import { useToaster } from '@harness/uicore'
import { useGet } from 'restful-react'
import { getErrorMessage } from 'utils/Utils'
import type { GitInfoProps } from 'utils/GitUtils'

interface UseDownloadRawFileParams extends Pick<GitInfoProps, 'repoMetadata' | 'gitRef' | 'resourcePath'> {
  filename: string
}

export function useDownloadRawFile() {
  const { error, response, refetch } = useGet({
    path: '',
    lazy: true
  })
  const [name, setName] = useState('')
  const { showError } = useToaster()
  const callback = useCallback(async () => {
    if (response) {
      const imageBlog = await response.blob()
      const imageURL = URL.createObjectURL(imageBlog)

      const anchor = document.createElement('a')
      anchor.href = imageURL
      anchor.download = name

      document.body.appendChild(anchor)
      anchor.click()
      // Cleaning up requires a timeout to work under Firefox
      setTimeout(() => {
        document.body.removeChild(anchor)
        URL.revokeObjectURL(imageURL)
      }, 100)
      return { status: true }
    }
  }, [name, response])

  useEffect(() => {
    if (error) {
      showError(getErrorMessage(error))
    } else if (response) {
      callback()
    }
  }, [error, showError, response, callback])

  return useCallback(
    ({ repoMetadata, resourcePath, gitRef, filename = 'download' }: UseDownloadRawFileParams) => {
      const rawURL = `/api/v1/repos/${repoMetadata?.path}/+/raw/${resourcePath}`
      setName(filename)
      refetch({ path: rawURL, queryParams: { git_ref: gitRef } })
    },
    [refetch]
  )
}
