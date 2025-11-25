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

import { getErrorInfoFromErrorObject, useToaster } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'

import { useAppStore } from './useAppStore'
import { useParentUtils } from './useParentUtils'

export function useDownloadArtifactFile() {
  const { scope } = useAppStore()
  const { getString } = useStrings()
  const { showError, clear } = useToaster()
  const { getCustomHeaders } = useParentUtils()

  const downloadFile = async (downloadURL: string, fileName: string) => {
    try {
      const response = await fetch(downloadURL, {
        method: 'GET',
        headers: {
          ...(getCustomHeaders ? getCustomHeaders() : {})
        }
      })
      const blob = await response.blob()
      // Create download link
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = fileName
      a.click()
      URL.revokeObjectURL(url)
    } catch (err) {
      clear()
      showError(getErrorInfoFromErrorObject(err as Error) ?? getString('failedToLoadData'))
    }
  }

  const openUrlInNewTab = async (downloadURL: string) => {
    window.open(downloadURL, '_blank')
  }

  const handleDownloadFile = async (repositoryIdentifier: string, path: string, fileName: string) => {
    const formattedFileName = encodeURIComponent(fileName)
    const downloadURL = window.getApiBaseUrl(`/har/api/v2/files/${formattedFileName}`)
    const downloadURLParams = new URLSearchParams({
      routingId: scope.accountId || '',
      account_identifier: scope.accountId || '',
      registry_identifier: repositoryIdentifier,
      path
    })
    const finalDownloadURL = `${downloadURL}?${downloadURLParams.toString()}`
    if (window.noAuthHeader) {
      return openUrlInNewTab(finalDownloadURL)
    }
    return downloadFile(finalDownloadURL, formattedFileName)
  }

  return { downloadFile: handleDownloadFile }
}
