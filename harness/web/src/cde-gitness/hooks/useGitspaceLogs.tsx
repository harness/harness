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

import { useGet } from 'restful-react'
import { useEffect } from 'react'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import { useAppContext } from 'AppContext'
import { useGetGitspaceInstanceLogs } from 'services/cde'

export const useGitspacesLogs = ({ gitspaceId }: { gitspaceId: string }) => {
  const { standalone } = useAppContext()

  const { accountIdentifier = '', orgIdentifier = '', projectIdentifier = '', space } = useGetCDEAPIParams()

  const gitness = useGet<any>({
    path: `api/v1/gitspaces/${space}/${gitspaceId}/+/logs/stream`,
    debounce: 500,
    lazy: true
  })

  const cde = useGetGitspaceInstanceLogs({
    projectIdentifier,
    orgIdentifier,
    accountIdentifier,
    gitspace_identifier: gitspaceId,
    lazy: true
  })

  useEffect(() => {
    if (!standalone) {
      cde.refetch()
    }
  }, [])

  return standalone ? gitness : cde
}
