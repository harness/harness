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

import React, { forwardRef, useCallback, useEffect } from 'react'
import * as Yup from 'yup'
import type { FormikProps } from 'formik'
import { FontVariation } from '@harnessio/design-system'
import {
  Container,
  Formik,
  Layout,
  Text,
  ThumbnailSelect,
  getErrorInfoFromErrorObject,
  useToaster
} from '@harnessio/uicore'
import { Anonymous, UserPassword, useCreateRegistryMutation } from '@harnessio/react-har-service-client'

import { useAppStore, useGetSpaceRef } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { decodeRef } from '@ar/hooks/useGetSpaceRef'
import { setFormikRef } from '@ar/common/utils'
import { REPO_KEY_REGEX, URL_REGEX } from '@ar/constants'
import { Separator } from '@ar/components/Separator/Separator'
import { RepositoryConfigType, type FormikFowardRef } from '@ar/common/types'
import {
  DockerRepositoryURLInputSource,
  UpstreamProxyAuthenticationMode,
  UpstreamProxyPackageType,
  UpstreamRegistry,
  UpstreamRegistryRequest
} from '@ar/pages/upstream-proxy-details/types'
import { UpstreamProxyPackageTypeList } from '@ar/pages/upstream-proxy-details/constants'

import CreateRepositoryWidget from '@ar/frameworks/RepositoryStep/CreateRepositoryWidget'
import repositoryFactory from '@ar/frameworks/RepositoryStep/RepositoryFactory'
import { getFormattedFormDataForCleanupPolicy } from '@ar/components/CleanupPolicyList/utils'
import type { RepositoryAbstractFactory } from '@ar/frameworks/RepositoryStep/RepositoryAbstractFactory'

import { getFormattedFormDataForAuthType } from './utils'
import css from './Forms.module.scss'

interface FormContentProps {
  readonly: boolean
  formikProps: FormikProps<UpstreamRegistryRequest>
  isEdit: boolean
  isPackageTypeReadonly?: boolean
  getDefaultValuesByRepositoryType: (
    type: UpstreamProxyPackageType,
    defaultValue: UpstreamRegistryRequest
  ) => UpstreamRegistryRequest
}

function FormContent(props: FormContentProps): JSX.Element {
  const { formikProps, getDefaultValuesByRepositoryType, isPackageTypeReadonly } = props
  const { getString } = useStrings()
  const { values } = formikProps
  const { packageType } = values

  useEffect(() => {
    const newDefaultValues = getDefaultValuesByRepositoryType(
      packageType as UpstreamProxyPackageType,
      {} as UpstreamRegistryRequest
    )
    formikProps.setValues(newDefaultValues)
  }, [packageType, getDefaultValuesByRepositoryType])

  return (
    <Container>
      <Layout.Vertical spacing="small">
        <Text font={{ variation: FontVariation.CARD_TITLE }}>
          {getString('repositoryDetails.repositoryForm.selectRepoType')}
        </Text>
        <Container>
          <ThumbnailSelect
            name="packageType"
            items={UpstreamProxyPackageTypeList.map(each => ({
              ...each,
              label: getString(each.label)
            }))}
            staticItems
            isReadonly={isPackageTypeReadonly}
          />
        </Container>
      </Layout.Vertical>
      {packageType && (
        <>
          <Separator topSeparation="var(--spacing-large)" bottomSeparation="var(--spacing-large)" />
          <CreateRepositoryWidget packageType={packageType} type={RepositoryConfigType.UPSTREAM} />
        </>
      )}
    </Container>
  )
}

interface UpstreamProxyCreateFormProps {
  isEdit: boolean
  setShowOverlay: (val: boolean) => void
  onSuccess: (data: UpstreamRegistry) => void
  defaultPackageType?: UpstreamProxyPackageType
  isPackageTypeReadonly?: boolean
  factory?: RepositoryAbstractFactory
}

function UpstreamProxyCreateForm(props: UpstreamProxyCreateFormProps, formikRef: FormikFowardRef): JSX.Element {
  const {
    isEdit,
    onSuccess,
    setShowOverlay,
    factory = repositoryFactory,
    defaultPackageType = UpstreamProxyPackageType.DOCKER,
    isPackageTypeReadonly = false
  } = props
  const { showSuccess, showError, clear } = useToaster()
  const { getString } = useStrings()
  const spaceRef = useGetSpaceRef('')
  const { parent, scope } = useAppStore()

  const { isLoading: loading, mutateAsync: createUpstreamProxy } = useCreateRegistryMutation()

  useEffect(() => {
    setShowOverlay(loading)
  }, [loading])

  const handleCreateUpstreamProxy = async (values: UpstreamRegistryRequest): Promise<void> => {
    try {
      const response = await createUpstreamProxy({
        queryParams: {
          space_ref: decodeRef(spaceRef)
        },
        body: {
          ...values,
          parentRef: decodeRef(spaceRef)
        }
      })
      if (response.content.status === 'SUCCESS') {
        clear()
        showSuccess(getString('upstreamProxyDetails.actions.createUpdateModal.createSuccessMessage'))
        onSuccess(response.content.data as UpstreamRegistry)
      }
    } catch (e: any) {
      showError(getErrorInfoFromErrorObject(e, true))
    }
  }

  const handleSubmit = (values: UpstreamRegistryRequest): void => {
    const repositoryType = factory.getRepositoryType(values.packageType)
    if (repositoryType) {
      const transfomedAuthType = getFormattedFormDataForAuthType(values, parent, scope)
      const transformedCleanupPolicy = getFormattedFormDataForCleanupPolicy(transfomedAuthType)
      const transformedValues = repositoryType.processUpstreamProxyFormData(
        transformedCleanupPolicy
      ) as UpstreamRegistryRequest
      handleCreateUpstreamProxy(transformedValues)
    }
  }

  const getDefaultValuesByRepositoryType = useCallback(
    (repoType: UpstreamProxyPackageType, defaultValue?: UpstreamRegistryRequest): UpstreamRegistryRequest => {
      const repositoryType = factory.getRepositoryType(repoType)
      if (repositoryType) {
        return repositoryType.getUpstreamProxyDefaultValues(defaultValue) as UpstreamRegistryRequest
      }
      return {} as UpstreamRegistryRequest
    },
    []
  )

  const getInitialValues = (): UpstreamRegistryRequest => {
    return getDefaultValuesByRepositoryType(defaultPackageType)
  }

  return (
    <Formik<UpstreamRegistryRequest>
      formName="upstream-proxy-create-form"
      onSubmit={handleSubmit}
      initialValues={getInitialValues()}
      validationSchema={Yup.object().shape({
        identifier: Yup.string()
          .required(getString('validationMessages.nameRequired'))
          .matches(REPO_KEY_REGEX, getString('validationMessages.repokeyRegExMessage')),
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
                })
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
        })
      })}>
      {(formik: FormikProps<UpstreamRegistryRequest>) => {
        setFormikRef(formikRef, formik)
        return (
          <Container className={css.formContainer}>
            <FormContent
              isEdit={isEdit}
              formikProps={formik}
              readonly={false}
              getDefaultValuesByRepositoryType={getDefaultValuesByRepositoryType}
              isPackageTypeReadonly={isPackageTypeReadonly}
            />
          </Container>
        )
      }}
    </Formik>
  )
}

export default forwardRef(UpstreamProxyCreateForm)
