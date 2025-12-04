/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import { useMemo } from 'react'
import { useAppStore } from './useAppStore'
interface UseGetDownloadFileURLParams {
  repositoryIdentifier: string
  fileName: string
  path: string
}

function getFormattedFileName(fileName: string) {
  return encodeURIComponent(fileName.replaceAll('/', '_'))
}

export function useGetDownloadFileURL(props: UseGetDownloadFileURLParams) {
  const { repositoryIdentifier, fileName, path } = props
  const { scope } = useAppStore()
  return useMemo(() => {
    try {
      const formattedFileName = getFormattedFileName(fileName)
      const downloadURL = window.getApiBaseUrl(`/har/api/v2/files/${formattedFileName}`)
      const downloadURLParams = new URLSearchParams({
        routingId: scope.accountId || '',
        account_identifier: scope.accountId || '',
        registry_identifier: repositoryIdentifier,
        path
      })
      return `${downloadURL}?${downloadURLParams.toString()}`
    } catch (error) {
      return ''
    }
  }, [fileName, path, repositoryIdentifier, scope.accountId])
}
