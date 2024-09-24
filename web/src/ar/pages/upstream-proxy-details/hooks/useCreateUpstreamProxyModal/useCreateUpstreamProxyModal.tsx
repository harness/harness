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
import { Button, ButtonVariation, Layout, ModalDialog } from '@harnessio/uicore'

import { useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import type { UpstreamProxyPackageType, UpstreamRegistry } from '@ar/pages/upstream-proxy-details/types'
import UpstreamProxyCreateForm from '@ar/pages/upstream-proxy-details/components/Forms/UpstreamProxyCreateForm'

interface useCreateUpstreamProxyModalProps {
  onSuccess: (data: UpstreamRegistry) => void
  defaultPackageType?: UpstreamProxyPackageType
  isPackageTypeReadonly?: boolean
}

export default function useCreateUpstreamProxyModal(props: useCreateUpstreamProxyModalProps) {
  const { onSuccess, defaultPackageType, isPackageTypeReadonly = false } = props
  const [showOverlay, setShowOverlay] = useState(false)
  const { getString } = useStrings()
  const { useModalHook } = useParentHooks()
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
        title={getString('upstreamProxyDetails.createForm.title')}
        footer={
          <Layout.Horizontal spacing="small">
            <Button
              variation={ButtonVariation.PRIMARY}
              type={'submit'}
              text={getString('upstreamProxyDetails.createForm.create')}
              data-id="upstreamProxy-save"
              onClick={handleSubmitForm}
              disabled={showOverlay}
            />
            <Button variation={ButtonVariation.TERTIARY} text={getString('cancel')} onClick={hideModal} />
          </Layout.Horizontal>
        }
        isCloseButtonShown
        width={850}
        showOverlay={showOverlay}>
        <UpstreamProxyCreateForm
          ref={stepRef}
          isEdit={false}
          setShowOverlay={setShowOverlay}
          defaultPackageType={defaultPackageType}
          isPackageTypeReadonly={isPackageTypeReadonly}
          onSuccess={data => {
            hideModal()
            onSuccess(data)
          }}
        />
      </ModalDialog>
    ),
    [showOverlay]
  )

  return [showModal, hideModal]
}
