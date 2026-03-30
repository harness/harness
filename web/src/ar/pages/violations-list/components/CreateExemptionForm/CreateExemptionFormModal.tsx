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
import { Button, ButtonVariation, Layout, useToaster } from '@harnessio/uicore'
import { Drawer, Position } from '@blueprintjs/core'
import { useCreateFirewallExceptionV3Mutation, type ArtifactScanV3 } from '@harnessio/react-har-service-client'

import { useAppStore } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'

import type { ExemptionFormSpec } from '../ExemptionForm/types'
import ExemptionForm from '../ExemptionForm/ExemptionForm'

import css from './CreateExemptionForm.module.scss'

interface CreateExemptionFormModalProps {
  data: ArtifactScanV3
  onClose: () => void
}

export default function CreateExemptionFormModal({ data, onClose }: CreateExemptionFormModalProps) {
  const { getString } = useStrings()
  const [submitting, setSubmitting] = useState(false)
  const { scope } = useAppStore()
  const { showSuccess, showError, clear } = useToaster()
  const formRef = React.useRef<FormikProps<unknown> | null>(null)

  const { mutateAsync: createExemption } = useCreateFirewallExceptionV3Mutation()

  const handleSubmitForm = (): void => {
    formRef.current?.submitForm()
  }

  const handleSubmit = async (values: ExemptionFormSpec) => {
    setSubmitting(true)
    return createExemption({
      body: {
        registryId: values.registryId,
        packageName: values.packageName,
        versionList: values.versionList.map(version => version.value as string),
        expireAfter: values.expireAfter,
        businessJustification: values.businessJustification,
        remediationPlan: values.remediationPlan
      },
      queryParams: {
        account_identifier: scope.accountId || ''
      }
    })
      .then(() => {
        clear()
        showSuccess(getString('violationsList.createExemptionForm.toasters.success'))
        onClose()
      })
      .catch(error => {
        clear()
        showError(
          error?.message || error?.error?.message || getString('violationsList.createExemptionForm.toasters.error')
        )
      })
      .finally(() => {
        setSubmitting(false)
      })
  }

  return (
    <Drawer
      className={css.drawerContainer}
      position={Position.RIGHT}
      isOpen
      isCloseButtonShown={false}
      size={'30%'}
      onClose={onClose}>
      <Layout.Vertical margin="medium">
        <Button minimal className={css.almostFullScreenCloseBtn} icon="cross" withoutBoxShadow onClick={onClose} />
        <ExemptionForm
          data={data}
          ref={formRef}
          onSubmit={handleSubmit}
          title={getString('violationsList.createExemptionForm.title')}
          subTitle={getString('violationsList.createExemptionForm.subTitle')}
        />
        <Layout.Horizontal spacing="small">
          <Button
            variation={ButtonVariation.PRIMARY}
            type={'submit'}
            text={getString('violationsList.createExemptionForm.actions.submit')}
            data-id="service-save"
            onClick={handleSubmitForm}
            disabled={submitting}
          />
          <Button
            variation={ButtonVariation.TERTIARY}
            text={getString('violationsList.createExemptionForm.actions.cancel')}
            onClick={onClose}
          />
        </Layout.Horizontal>
      </Layout.Vertical>
    </Drawer>
  )
}
