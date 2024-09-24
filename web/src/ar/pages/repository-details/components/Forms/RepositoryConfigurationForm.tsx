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

import React, { forwardRef, useContext } from 'react'
import * as Yup from 'yup'
import { useQueryClient } from '@tanstack/react-query'
import { Formik, FormikForm, getErrorInfoFromErrorObject, useToaster } from '@harnessio/uicore'
import { RegistryRequestRequestBody, useModifyRegistryMutation } from '@harnessio/react-har-service-client'

import { useGetSpaceRef } from '@ar/hooks'
import { setFormikRef } from '@ar/common/utils'
import { useStrings } from '@ar/frameworks/strings'
import repositoryFactory from '@ar/frameworks/RepositoryStep/RepositoryFactory'
import {
  getFormattedFormDataForCleanupPolicy,
  getFormattedIntialValuesForCleanupPolicy
} from '@ar/components/CleanupPolicyList/utils'
import { RepositoryProviderContext } from '@ar/pages/repository-details/context/RepositoryProvider'
import RepositoryConfigurationFormContent from '@ar/pages/repository-details/components/FormContent/RepositoryConfigurationFormContent'

import type { FormikFowardRef } from '@ar/common/types'
import type { VirtualRegistry, VirtualRegistryRequest } from '@ar/pages/repository-details/types'
import type { RepositoryAbstractFactory } from '@ar/frameworks/RepositoryStep/RepositoryAbstractFactory'

import css from './RepositoryDetailsForm.module.scss'

interface RepositoryConfigurationFormProps {
  readonly: boolean
  factory?: RepositoryAbstractFactory
}

function RepositoryConfigurationForm(props: RepositoryConfigurationFormProps, formikRef: FormikFowardRef): JSX.Element {
  const { readonly, factory = repositoryFactory } = props
  const { data, setIsUpdating } = useContext(RepositoryProviderContext)
  const { showSuccess, showError, clear } = useToaster()
  const { getString } = useStrings()
  const spaceRef = useGetSpaceRef()
  const queryClient = useQueryClient()

  const { mutateAsync: modifyRepository } = useModifyRegistryMutation()

  const handleModifyRepository = async (values: VirtualRegistryRequest): Promise<void> => {
    try {
      setIsUpdating(true)
      const response = await modifyRepository({
        registry_ref: spaceRef,
        body: values as unknown as RegistryRequestRequestBody
      })
      if (response.content.status === 'SUCCESS') {
        clear()
        showSuccess(getString('repositoryDetails.repositoryForm.repositoryUpdated'))
        queryClient.invalidateQueries(['GetRegistry'])
      }
    } catch (e: any) {
      showError(getErrorInfoFromErrorObject(e, true))
    } finally {
      setIsUpdating(false)
    }
  }

  const handleSubmit = async (values: VirtualRegistryRequest): Promise<void> => {
    const repositoryType = factory.getRepositoryType(values.packageType)
    if (repositoryType) {
      const transformedValues = repositoryType.processRepositoryFormData(values) as VirtualRegistryRequest
      const formattedValuesForCleanupPolicy = getFormattedFormDataForCleanupPolicy(transformedValues)
      await handleModifyRepository(formattedValuesForCleanupPolicy)
    }
  }

  const getInitialData = (repoData: VirtualRegistry): VirtualRegistryRequest => {
    const repositoryType = factory.getRepositoryType(repoData.packageType)
    if (repositoryType) {
      const initialValues = repositoryType.getRepositoryInitialValues(repoData) as VirtualRegistryRequest
      const transformedInitialValuesForCleanupPolicy = getFormattedIntialValuesForCleanupPolicy(initialValues)
      return transformedInitialValuesForCleanupPolicy
    }
    return {} as VirtualRegistryRequest
  }

  return (
    <Formik<VirtualRegistryRequest>
      onSubmit={values => {
        handleSubmit(values)
      }}
      validationSchema={Yup.object().shape({
        cleanupPolicy: Yup.array()
          .of(
            Yup.object().shape({
              name: Yup.string().trim().required(getString('validationMessages.cleanupPolicy.nameRequired')),
              expireDays: Yup.number()
                .required(getString('validationMessages.cleanupPolicy.expireDaysRequired'))
                .positive(getString('validationMessages.cleanupPolicy.positiveExpireDays'))
            })
          )
          .optional()
          .nullable()
      })}
      formName="registry-form"
      initialValues={getInitialData(data as VirtualRegistry)}>
      {formik => {
        setFormikRef(formikRef, formik)
        return (
          <FormikForm className={css.formContainer}>
            <RepositoryConfigurationFormContent readonly={readonly} />
          </FormikForm>
        )
      }}
    </Formik>
  )
}

export default forwardRef(RepositoryConfigurationForm)
