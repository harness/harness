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
import { ButtonVariation } from '@harnessio/uicore'

import { useParentComponents } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import type { RepositoryPackageType } from '@ar/common/types'
import type { Repository } from '@ar/pages/repository-details/types'
import type { RbacButtonProps } from '@ar/__mocks__/components/RbacButton'
import { useCreateRepositoryModal } from '@ar/pages/repository-details/hooks/useCreateRepositoryModal/useCreateRepositoryModal'

import '@ar/pages/version-details/VersionFactory'
import '@ar/pages/repository-details/RepositoryFactory'

interface CreateRegistryButtonProps extends RbacButtonProps {
  onSuccess: (data: Repository) => void
  allowedPackageTypes?: RepositoryPackageType[]
}

export default function CreateRegistryButton(props: CreateRegistryButtonProps) {
  const { onSuccess, allowedPackageTypes, ...rest } = props
  const { RbacButton } = useParentComponents()
  const { getString } = useStrings()

  const [show, hide] = useCreateRepositoryModal({
    onSuccess: data => {
      hide()
      onSuccess(data)
    },
    allowedPackageTypes
  })

  return (
    <RbacButton
      variation={ButtonVariation.SECONDARY}
      icon={'plus'}
      iconProps={{ size: 10 }}
      text={getString('repositoryList.newRegistry')}
      {...rest}
      onClick={() => {
        show()
      }}
    />
  )
}
