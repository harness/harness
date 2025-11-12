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

import { useAppStore } from './useAppStore'

interface UseGetDownloadFileURLParams {
  repositoryIdentifier: string
  fileName: string
  path: string
}

export function useGetDownloadFileURL(props: UseGetDownloadFileURLParams) {
  const { repositoryIdentifier, fileName, path } = props
  const { scope } = useAppStore()
  const formattedFileName = encodeURIComponent(fileName)
  const downloadURL = window.getApiBaseUrl(`/har/api/v2/files/${formattedFileName}`)
  const downloadURLParams = new URLSearchParams({
    account_identifier: scope.accountId || '',
    registry_identifier: repositoryIdentifier,
    path
  })

  return `${downloadURL}?${downloadURLParams.toString()}`
}
