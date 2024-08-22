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
import type { TypesGitspaceEventResponse } from 'cde-gitness/services'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import { useAppContext } from 'AppContext'
import { useGetGitspaceEvents } from 'services/cde'

export const useGitspaceEvents = ({ gitspaceId }: { gitspaceId: string }) => {
  const { standalone } = useAppContext()

  const { accountIdentifier = '', orgIdentifier = '', projectIdentifier = '', space } = useGetCDEAPIParams()

  const gitness = useGet<TypesGitspaceEventResponse[]>({
    path: `/api/v1/gitspaces/${space}/${gitspaceId}/+/events`,
    debounce: 500,
    lazy: true
  })

  const cde = useGetGitspaceEvents({
    accountIdentifier,
    orgIdentifier,
    projectIdentifier,
    gitspace_identifier: gitspaceId,
    lazy: true
  })

  useEffect(() => {
    if (standalone) {
      gitness.refetch()
    } else {
      cde.refetch()
    }
  }, [])

  return standalone ? gitness : cde
}
