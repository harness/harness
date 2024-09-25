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

import React, { useState } from 'react'
import type { FormikProps } from 'formik'
import { Color, FontVariation } from '@harnessio/design-system'
import { Button, ButtonVariation, Layout, ModalDialog, Text } from '@harnessio/uicore'

import { useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import type { RepositoryPackageType } from '@ar/common/types'
import RepositoryCreateForm from '@ar/pages/repository-details/components/Forms/RepositoryCreateForm'
import type { Repository } from '@ar/pages/repository-details/types'

interface useCreateRepositoryModalProps {
  onSuccess: (data: Repository) => void
  allowedPackageTypes?: RepositoryPackageType[]
}

export function useCreateRepositoryModal(props: useCreateRepositoryModalProps) {
  const { onSuccess, allowedPackageTypes } = props
  const { getString } = useStrings()
  const { useModalHook } = useParentHooks()
  const [showOverlay, setShowOverlay] = useState(false)
  const stepRef = React.useRef<FormikProps<unknown> | null>(null)

  const handleSubmitForm = (evt: React.MouseEvent<Element, MouseEvent>): void => {
    evt.preventDefault()
    stepRef.current?.submitForm()
  }

  const [showModal, hideModal] = useModalHook(
    () => (
      <ModalDialog
        isOpen={true}
        enforceFocus={false}
        canEscapeKeyClose
        canOutsideClickClose
        onClose={() => {
          hideModal()
        }}
        title={
          <>
            <Text font={{ variation: FontVariation.H3 }} margin={{ bottom: 'small' }}>
              {getString('repositoryDetails.repositoryForm.modalTitle')}
            </Text>
            <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_500}>
              {getString('repositoryDetails.repositoryForm.modalSubTitle')}
            </Text>
          </>
        }
        isCloseButtonShown
        width={800}
        footer={
          <Layout.Horizontal spacing="small">
            <Button
              variation={ButtonVariation.PRIMARY}
              type={'submit'}
              text={getString('repositoryDetails.repositoryForm.create')}
              data-id="service-save"
              onClick={handleSubmitForm}
              disabled={showOverlay}
            />
            <Button variation={ButtonVariation.TERTIARY} text={getString('cancel')} onClick={hideModal} />
          </Layout.Horizontal>
        }>
        <RepositoryCreateForm
          allowedPackageTypes={allowedPackageTypes}
          onSuccess={onSuccess}
          setShowOverlay={setShowOverlay}
          ref={stepRef}
        />
      </ModalDialog>
    ),
    [showOverlay]
  )

  return [showModal, hideModal]
}
