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

import React from 'react'
import { useHistory } from 'react-router-dom'

import { useRoutes } from '@ar/hooks'
import { RepositoryConfigType } from '@ar/common/types'
import type { Repository } from '@ar/pages/repository-details/types'
import { RepositoryDetailsTab } from '@ar/pages/repository-details/constants'
import { useCreateRepositoryModal } from '@ar/pages/repository-details/hooks/useCreateRepositoryModal/useCreateRepositoryModal'
import useCreateUpstreamProxyModal from '@ar/pages/upstream-proxy-details/hooks/useCreateUpstreamProxyModal/useCreateUpstreamProxyModal'
import CreateRepositoryButton from './CreateRepositoryButton'

export function CreateRepository(): JSX.Element {
  const history = useHistory()
  const routes = useRoutes()

  const handleRedirectToRepoDetails = (data: Repository): void => {
    history.push(
      routes.toARRepositoryDetails({
        repositoryIdentifier: data.identifier,
        tab: RepositoryDetailsTab.CONFIGURATION
      })
    )
  }

  const [showCreateRepositoryModal] = useCreateRepositoryModal({
    onSuccess: handleRedirectToRepoDetails
  })

  const [showCreateUpstreamProxyModal] = useCreateUpstreamProxyModal({
    onSuccess: handleRedirectToRepoDetails
  })

  const handleClickCreateRepositoryButton = (type: RepositoryConfigType): void => {
    if (type === RepositoryConfigType.VIRTUAL) {
      showCreateRepositoryModal()
    } else {
      showCreateUpstreamProxyModal()
    }
  }

  return (
    <>
      <CreateRepositoryButton onClick={handleClickCreateRepositoryButton} />
    </>
  )
}
