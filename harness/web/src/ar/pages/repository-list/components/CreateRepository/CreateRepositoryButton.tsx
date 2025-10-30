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
import { defaultTo } from 'lodash-es'
import { ButtonVariation, SplitButton } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { useAppStore, useParentHooks } from '@ar/hooks'
import { RepositoryConfigType } from '@ar/common/types'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'

import ButtonOption from './ButtonOption'
import css from './CreateRepository.module.scss'

interface CreateRepositoryButtonProps {
  onClick: (type: RepositoryConfigType) => void
  variation?: ButtonVariation
}

export default function CreateRepositoryButton(props: CreateRepositoryButtonProps): JSX.Element {
  const { variation, onClick } = props
  const { usePermission } = useParentHooks()
  const { scope } = useAppStore()
  const { getString } = useStrings()

  // CHANGE_ME: update permissions once we get actual premission for AR module
  const [canDoAction] = usePermission({
    permissions: [PermissionIdentifier.EDIT_ARTIFACT_REGISTRY],
    resourceScope: {
      accountIdentifier: scope.accountId,
      orgIdentifier: scope.orgIdentifier,
      projectIdentifier: scope.projectIdentifier
    },
    resource: {
      resourceType: ResourceType.ARTIFACT_REGISTRY
    }
  })

  return (
    <SplitButton
      variation={defaultTo(variation, ButtonVariation.PRIMARY)}
      text={getString('repositoryList.newRepository')}
      icon={'plus'}
      iconProps={{ size: 10 }}
      onClick={() => onClick(RepositoryConfigType.VIRTUAL)}
      disabled={!canDoAction}
      dropdownDisabled={!canDoAction}
      popoverProps={{
        className: css.splitButton
      }}>
      <ButtonOption
        className={css.option}
        icon="virtual-registry"
        iconProps={{ size: 16 }}
        text={getString('repositoryList.artifactRegistry.label')}
        subText={getString('repositoryList.artifactRegistry.subLabel')}
        onClick={() => onClick(RepositoryConfigType.VIRTUAL)}
      />
      <ButtonOption
        className={css.option}
        icon="upstream-registry"
        iconProps={{ size: 16 }}
        text={getString('repositoryList.upstreamProxy.label')}
        subText={getString('repositoryList.upstreamProxy.subLabel')}
        onClick={() => onClick(RepositoryConfigType.UPSTREAM)}
      />
    </SplitButton>
  )
}
