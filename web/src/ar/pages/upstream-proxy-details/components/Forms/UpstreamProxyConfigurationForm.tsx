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
import { Formik, FormikForm, getErrorInfoFromErrorObject, useToaster } from '@harnessio/uicore'
import { Anonymous, UserPassword, useModifyRegistryMutation } from '@harnessio/react-har-service-client'

import { URL_REGEX } from '@ar/constants'
import { useAppStore, useGetSpaceRef } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { queryClient } from '@ar/utils/queryClient'
import type { FormikFowardRef } from '@ar/common/types'
import { setFormikRef } from '@ar/common/utils'

import repositoryFactory from '@ar/frameworks/RepositoryStep/RepositoryFactory'
import { RepositoryProviderContext } from '@ar/pages/repository-details/context/RepositoryProvider'
import type { RepositoryAbstractFactory } from '@ar/frameworks/RepositoryStep/RepositoryAbstractFactory'
import {
  getFormattedFormDataForCleanupPolicy,
  getFormattedIntialValuesForCleanupPolicy
} from '@ar/components/CleanupPolicyList/utils'

import UpstreamProxyConfigurationFormContent from '../FormContent/UpstreamProxyConfigurationFormContent'
import {
  DockerRepositoryURLInputSource,
  UpstreamProxyAuthenticationMode,
  type UpstreamRegistry,
  type UpstreamRegistryRequest
} from '../../types'
import { getFormattedFormDataForAuthType } from './utils'

import css from './Forms.module.scss'

interface UpstreamProxyConfigurationFormProps {
  readonly: boolean
  factory?: RepositoryAbstractFactory
}

function UpstreamProxyConfigurationForm(
  props: UpstreamProxyConfigurationFormProps,
  formikRef: FormikFowardRef
): JSX.Element {
  const { readonly, factory = repositoryFactory } = props
  const { data, setIsUpdating } = useContext(RepositoryProviderContext)
  const { showSuccess, showError, clear } = useToaster()
  const { getString } = useStrings()
  const { parent } = useAppStore()
  const spaceRef = useGetSpaceRef()

  const { mutateAsync: modifyUpstreamProxy } = useModifyRegistryMutation()

  const getInitialValues = (repoData: UpstreamRegistry): UpstreamRegistryRequest => {
    const repositoryType = factory.getRepositoryType(repoData.packageType)
    if (repositoryType) {
      const transformedInitialValuesForCleanupPolicy = getFormattedIntialValuesForCleanupPolicy(repoData)
      return repositoryType.getUpstreamProxyInitialValues(
        transformedInitialValuesForCleanupPolicy
      ) as UpstreamRegistryRequest
    }
    return {} as UpstreamRegistryRequest
  }

  const handleModifyUpstreamProxy = async (values: UpstreamRegistryRequest): Promise<void> => {
    try {
      setIsUpdating(true)
      const response = await modifyUpstreamProxy({
        registry_ref: spaceRef,
        body: values
      })
      if (response.content.status === 'SUCCESS') {
        clear()
        showSuccess(getString('upstreamProxyDetails.actions.createUpdateModal.updateSuccessMessage'))
        queryClient.invalidateQueries(['GetRegistry'])
      }
    } catch (e: any) {
      showError(getErrorInfoFromErrorObject(e, true))
    } finally {
      setIsUpdating(false)
    }
  }

  const handleSubmit = async (values: UpstreamRegistryRequest): Promise<void> => {
    const repositoryType = factory.getRepositoryType(values.packageType)
    if (repositoryType) {
      const transfomedAuthType = getFormattedFormDataForAuthType(values, parent)
      const transformedCleanupPolicy = getFormattedFormDataForCleanupPolicy(transfomedAuthType)
      const transformedValues = repositoryType.processUpstreamProxyFormData(
        transformedCleanupPolicy
      ) as UpstreamRegistryRequest
      await handleModifyUpstreamProxy(transformedValues)
    }
  }

  return (
    <Formik<UpstreamRegistryRequest>
      onSubmit={values => {
        handleSubmit(values)
      }}
      formName="upstream-repository-form"
      initialValues={getInitialValues(data as UpstreamRegistry)}
      validationSchema={Yup.object().shape({
        config: Yup.object().shape({
          authType: Yup.string()
            .required()
            .oneOf([UpstreamProxyAuthenticationMode.ANONYMOUS, UpstreamProxyAuthenticationMode.USER_NAME_AND_PASSWORD]),
          auth: Yup.object()
            .when(['authType'], {
              is: (authType: UpstreamProxyAuthenticationMode) =>
                authType === UpstreamProxyAuthenticationMode.USER_NAME_AND_PASSWORD,
              then: (schema: Yup.ObjectSchema<UserPassword | Anonymous>) =>
                schema.shape({
                  userName: Yup.string().trim().required(getString('validationMessages.userNameRequired')),
                  secretIdentifier: Yup.string().trim().required(getString('validationMessages.passwordRequired'))
                }),
              otherwise: Yup.object().optional().nullable()
            })
            .nullable(),
          url: Yup.string().when(['source'], {
            is: (source: DockerRepositoryURLInputSource) => source === DockerRepositoryURLInputSource.Custom,
            then: (schema: Yup.StringSchema) =>
              schema
                .trim()
                .required(getString('validationMessages.urlRequired'))
                .matches(URL_REGEX, getString('validationMessages.urlPattern')),
            otherwise: (schema: Yup.StringSchema) => schema.trim().notRequired()
          })
        }),
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
      })}>
      {formik => {
        setFormikRef(formikRef, formik)
        return (
          <FormikForm className={css.formContainer}>
            <UpstreamProxyConfigurationFormContent formikProps={formik} readonly={readonly} />
          </FormikForm>
        )
      }}
    </Formik>
  )
}

export default forwardRef(UpstreamProxyConfigurationForm)
