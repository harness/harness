/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { useParams } from 'react-router-dom'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from '../../hooks/useGetSpaceParam'

export interface CDEPathParams {
  accountIdentifier?: string
  orgIdentifier?: string
  projectIdentifier?: string
  space?: string
}

interface UseGetCDEAPIParamsProps {
  gitspacePath?: string
  fromUsageDashboard?: boolean
}

export const useGetCDEAPIParams = (props?: UseGetCDEAPIParamsProps): CDEPathParams => {
  const { gitspacePath, fromUsageDashboard = false } = props || {}
  const space = useGetSpaceParam()
  const { standalone, accountInfo } = useAppContext()
  const {
    accountId: accountIdentifier,
    orgIdentifier,
    projectIdentifier
  } = useParams<{
    accountId: string
    orgIdentifier: string
    projectIdentifier: string
  }>()

  // When fromUsageDashboard=true and gitspacePath is provided, extract identifiers from gitspacePath
  if (fromUsageDashboard && gitspacePath) {
    const pathParts = gitspacePath.split('/') || []
    const gitspaceAccountIdentifier = pathParts.length >= 1 ? pathParts[0] : ''
    const gitspaceOrgIdentifier = pathParts.length >= 2 ? pathParts[1] : ''
    const gitspaceProjectIdentifier = pathParts.length >= 3 ? pathParts[2] : ''

    return {
      accountIdentifier: gitspaceAccountIdentifier || accountInfo?.identifier || accountIdentifier,
      orgIdentifier: gitspaceOrgIdentifier || orgIdentifier,
      projectIdentifier: gitspaceProjectIdentifier || projectIdentifier,
      space: gitspacePath
    }
  }

  return standalone
    ? { space }
    : {
        accountIdentifier: accountIdentifier || accountInfo?.identifier,
        orgIdentifier,
        projectIdentifier,
        space
      }
}
