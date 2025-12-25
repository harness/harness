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
import { FontVariation } from '@harnessio/design-system'
import { Button, ButtonVariation, Layout, ModalDialog, Text } from '@harnessio/uicore'

import { useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'

import type { TypesLabel, TypesLabelWithValues } from 'services/code'

import CreateLabelForm from '../components/LabelForm/CreateLabelForm'

import css from './styles.module.scss'

export interface useCreateLabelModalProps {
  onSuccess: (response: TypesLabelWithValues) => void
  data?: TypesLabel
}

export function useCreateLabelModal(props: useCreateLabelModalProps) {
  const { onSuccess } = props
  const { getString } = useStrings()
  const { useModalHook } = useParentHooks()
  const [showOverlay, setShowOverlay] = useState(false)
  const formRef = React.useRef<FormikProps<unknown> | null>(null)

  const handleSubmitForm = (evt: React.MouseEvent<Element, MouseEvent>): void => {
    evt.preventDefault()
    formRef.current?.submitForm()
  }

  const [showModal, hideModal] = useModalHook(
    () => (
      <ModalDialog
        isOpen={true}
        enforceFocus={false}
        showOverlay={showOverlay}
        className={css.labelsModal}
        canEscapeKeyClose
        canOutsideClickClose
        onClose={() => {
          hideModal()
        }}
        title={
          <Text font={{ variation: FontVariation.H3 }} margin={{ bottom: 'small' }}>
            {getString('labelsList.createLabelModal.title')}
          </Text>
        }
        isCloseButtonShown
        width={900}
        height={700}
        footer={
          <Layout.Horizontal spacing="small">
            <Button
              variation={ButtonVariation.PRIMARY}
              type="submit"
              text={getString('labelsList.createLabelModal.actions.create')}
              data-id="label-save"
              onClick={handleSubmitForm}
              disabled={showOverlay}
            />
            <Button
              variation={ButtonVariation.TERTIARY}
              text={getString('labelsList.createLabelModal.actions.cancel')}
              onClick={hideModal}
            />
          </Layout.Horizontal>
        }>
        <CreateLabelForm
          ref={formRef}
          onSuccess={response => {
            onSuccess(response)
            hideModal()
          }}
          setShowOverlay={setShowOverlay}
        />
      </ModalDialog>
    ),
    [showOverlay, onSuccess]
  )

  return [showModal, hideModal]
}
