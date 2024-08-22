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
import { Drawer, Position } from '@blueprintjs/core'
import { Button } from '@harnessio/uicore'

import { useParentHooks } from '@ar/hooks'
import type { RepositoryPackageType } from '@ar/common/types'
import RepositorySetupClientWidget from '@ar/frameworks/RepositoryStep/RepositorySetupClientWidget'
import type { RepositoySetupClientProps } from '@ar/frameworks/RepositoryStep/Repository'
import css from './useSetupClientModal.module.scss'

export interface useSetupClientModalProps extends Omit<RepositoySetupClientProps, 'onClose'> {
  packageType: RepositoryPackageType
}

export function useSetupClientModal(props: useSetupClientModalProps) {
  const { packageType, repoKey, artifactKey, versionKey } = props
  const { useModalHook } = useParentHooks()

  const [showModal, hideModal] = useModalHook(() => (
    <Drawer position={Position.RIGHT} isOpen={true} isCloseButtonShown={false} size={'50%'} onClose={hideModal}>
      <Button minimal className={css.almostFullScreenCloseBtn} icon="cross" withoutBoxShadow onClick={hideModal} />
      <RepositorySetupClientWidget
        repoKey={repoKey}
        artifactKey={artifactKey}
        versionKey={versionKey}
        onClose={hideModal}
        type={packageType as RepositoryPackageType}
      />
    </Drawer>
  ))

  return [showModal, hideModal]
}
