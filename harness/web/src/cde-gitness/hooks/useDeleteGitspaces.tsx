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

import { useMutate } from 'restful-react'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import { useAppContext } from 'AppContext'
import { useDeleteGitspace } from 'services/cde'

interface UseDeleteGitspacesProps {
  gitspaceId: string
  gitspacePath?: string
  fromUsageDashboard?: boolean
}

export const useDeleteGitspaces = ({
  gitspaceId,
  gitspacePath,
  fromUsageDashboard = false
}: UseDeleteGitspacesProps) => {
  const { standalone } = useAppContext()

  const {
    accountIdentifier = '',
    orgIdentifier = '',
    projectIdentifier = '',
    space
  } = useGetCDEAPIParams({
    gitspacePath,
    fromUsageDashboard
  })

  const gitness = useMutate<any>({
    verb: 'DELETE',
    path: `/api/v1/gitspaces/${space}/${gitspaceId}/+`
  })

  const cde = useDeleteGitspace({
    projectIdentifier,
    orgIdentifier,
    accountIdentifier
  })

  return standalone ? gitness : cde
}
