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

import React, { forwardRef, type PropsWithChildren } from 'react'
import { getErrorInfoFromErrorObject, useToaster } from '@harnessio/uicore'

import { type TypesLabelWithValues, useSaveSpaceLabel } from 'services/code'

import { useGetSpaceRef } from '@ar/hooks'
import type { FormikFowardRef } from '@ar/common/types'

import LabelForm from './LabelForm'
import type { LabelFormData } from './types'

interface CreateLabelFormProps {
  onSuccess: (response: TypesLabelWithValues) => void
  setShowOverlay: (show: boolean) => void
}

function CreateLabelForm(props: PropsWithChildren<CreateLabelFormProps>, ref: FormikFowardRef) {
  const { setShowOverlay } = props
  const spaceRef = useGetSpaceRef()
  const { showError } = useToaster()

  const { mutate: saveLabel } = useSaveSpaceLabel({
    space_ref: spaceRef
  })

  const handleCreateLabel = async (formData: LabelFormData) => {
    const { labelValues, ...rest } = formData
    try {
      setShowOverlay(true)
      const response = await saveLabel({
        values: labelValues,
        label: rest
      })
      props.onSuccess(response)
    } catch (e: any) {
      showError(getErrorInfoFromErrorObject(e, true))
    } finally {
      setShowOverlay(false)
    }
  }
  return <LabelForm isEdit={false} onSubmit={handleCreateLabel} ref={ref} />
}

export default forwardRef(CreateLabelForm)
