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
import type { TypesGitspaceConfig } from 'cde-gitness/services'
import { LIST_FETCHING_LIMIT } from 'utils/Utils'
import { useGetCDEAPIParams } from 'cde-gitness/hooks/useGetCDEAPIParams'
import { useAppContext } from 'AppContext'
import { useListGitspaces } from 'services/cde'

export const useLisitngApi = ({ page }: { page: number }) => {
  const { standalone } = useAppContext()

  const { accountIdentifier = '', orgIdentifier = '', projectIdentifier = '', space } = useGetCDEAPIParams()

  const gitness = useGet<TypesGitspaceConfig[]>({
    path: `/api/v1/spaces/${space}/+/gitspaces`,
    queryParams: { page, limit: LIST_FETCHING_LIMIT },
    debounce: 500,
    lazy: true
  })

  // const cde = useGet<TypesGitspaceConfig[]>({
  //   path: `/cde/api/v1/accounts/${accountIdentifier}/orgs/${orgIdentifier}/projects/${projectIdentifier}/gitspaces`,
  //   queryParams: { page, limit: LIST_FETCHING_LIMIT },
  //   lazy: true
  // })

  const cde = useListGitspaces({
    queryParams: { page, limit: LIST_FETCHING_LIMIT },
    accountIdentifier,
    orgIdentifier,
    projectIdentifier,
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
