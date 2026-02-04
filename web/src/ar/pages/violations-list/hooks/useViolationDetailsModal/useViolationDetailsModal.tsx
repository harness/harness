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
import ViolationDetailsContent from '../../components/ViolationDetailsContent/ViolationDetailsContent'

import css from '@ar/pages/repository-details/hooks/useSetupClientModal/useSetupClientModal.module.scss'

export interface useViolationDetailsModalProps {
  scanId: string
  onClose?: () => void
}

export function useViolationDetailsModal(props: useViolationDetailsModalProps) {
  const { onClose } = props
  const { useModalHook } = useParentHooks()

  const [showModal, hideModal] = useModalHook(() => {
    const handleCloseModal = () => {
      onClose?.()
      hideModal()
    }
    return (
      <Drawer
        position={Position.RIGHT}
        isOpen={true}
        isCloseButtonShown={false}
        size={'30%'}
        onClose={handleCloseModal}>
        <Button
          minimal
          className={css.almostFullScreenCloseBtn}
          icon="cross"
          withoutBoxShadow
          onClick={handleCloseModal}
        />
        <ViolationDetailsContent scanId={props.scanId} onClose={handleCloseModal} />
      </Drawer>
    )
  })

  return [showModal, hideModal]
}
