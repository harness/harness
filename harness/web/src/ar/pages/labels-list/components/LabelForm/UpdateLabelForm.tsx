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

import React, { forwardRef, useMemo, type PropsWithChildren } from 'react'
import { Spinner } from '@blueprintjs/core'
import { getErrorInfoFromErrorObject, PageError, useToaster } from '@harnessio/uicore'

import { ColorName } from 'utils/Utils'
import { type TypesLabel, type TypesLabelWithValues, useListSpaceLabelValues, useSaveSpaceLabel } from 'services/code'

import { useGetSpaceRef } from '@ar/hooks'
import type { FormikFowardRef } from '@ar/common/types'

import LabelForm from './LabelForm'
import type { LabelFormData } from './types'

interface UpdateLabelFormProps {
  data: TypesLabel
  onSuccess: (response: TypesLabelWithValues) => void
  setShowOverlay: (show: boolean) => void
}

function UpdateLabelForm(props: PropsWithChildren<UpdateLabelFormProps>, ref: FormikFowardRef) {
  const { setShowOverlay, data } = props
  const spaceRef = useGetSpaceRef()
  const { showError } = useToaster()

  const { mutate: updateLabel } = useSaveSpaceLabel({
    space_ref: spaceRef
  })

  const {
    data: valuesList,
    loading,
    error,
    refetch
  } = useListSpaceLabelValues({
    space_ref: spaceRef,
    key: data?.key || ''
  })

  const initialValues: LabelFormData = useMemo(() => {
    const labelValues =
      valuesList?.map(value => ({
        id: value?.id || 0,
        value: value?.value || '',
        color: value?.color || ColorName.Blue
      })) || []
    return {
      id: data?.id || 0,
      key: data?.key || '',
      description: data?.description || '',
      scope: data?.scope || 0,
      type: data?.type || 'static',
      color: data?.color || ColorName.Blue,
      labelValues
    }
  }, [data, valuesList])

  const handleUpdateLabel = async (formData: LabelFormData) => {
    const { labelValues, ...rest } = formData
    try {
      setShowOverlay(true)
      const response = await updateLabel({
        label: rest,
        values: labelValues
      })
      props.onSuccess(response)
    } catch (e: any) {
      showError(getErrorInfoFromErrorObject(e, true))
    } finally {
      setShowOverlay(false)
    }
  }
  if (loading) return <Spinner size={Spinner.SIZE_SMALL} />
  if (error) return <PageError message={getErrorInfoFromErrorObject(error)} onClick={() => refetch()} />
  return <LabelForm isEdit={true} initialValues={initialValues} onSubmit={handleUpdateLabel} ref={ref} />
}

export default forwardRef(UpdateLabelForm)
