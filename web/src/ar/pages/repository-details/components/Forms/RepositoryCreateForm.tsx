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

import React, { useCallback, useEffect } from 'react'
import type { FormikProps } from 'formik'
import * as Yup from 'yup'
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
import { RegistryRequestRequestBody, useCreateRegistryMutation } from '@harnessio/react-har-service-client'

import type { RepositoryAbstractFactory } from '@ar/frameworks/RepositoryStep/RepositoryAbstractFactory'
import repositoryFactory from '@ar/frameworks/RepositoryStep/RepositoryFactory'
import CreateRepositoryWidget from '@ar/frameworks/RepositoryStep/CreateRepositoryWidget'

import { REPO_KEY_REGEX } from '@ar/constants'
import { useStrings } from '@ar/frameworks/strings'
import { decodeRef } from '@ar/hooks/useGetSpaceRef'
import { useGetSpaceRef } from '@ar/hooks'
import type { FormikFowardRef } from '@ar/common/types'
import { RepositoryPackageType, RepositoryConfigType } from '@ar/common/types'
import { setFormikRef } from '@ar/common/utils'
import { RepositoryTypes } from '@ar/common/constants'
import { Separator } from '@ar/components/Separator/Separator'
import type { Repository, VirtualRegistryRequest } from '@ar/pages/repository-details/types'
import { getFormattedFormDataForCleanupPolicy } from '@ar/components/CleanupPolicyList/utils'

import css from './RepositoryDetailsForm.module.scss'

interface FormContentProps {
  formikProps: FormikProps<VirtualRegistryRequest>
  allowedPackageTypes?: RepositoryPackageType[]
  getDefaultValuesByRepositoryType: (
    type: RepositoryPackageType,
    defaultValue: VirtualRegistryRequest
  ) => VirtualRegistryRequest
}

function FormContent(props: FormContentProps): JSX.Element {
  const { formikProps, getDefaultValuesByRepositoryType, allowedPackageTypes } = props
  const { getString } = useStrings()
  const { values } = formikProps
  const { packageType, config } = values
  const { type } = config

  useEffect(() => {
    const newDefaultValues = getDefaultValuesByRepositoryType(packageType as RepositoryPackageType, values)
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
            items={RepositoryTypes.map(each => ({
              ...each,
              label: getString(each.label),
              disabled: allowedPackageTypes?.length ? !allowedPackageTypes.includes(each.value) : each.disabled
            }))}
            staticItems
          />
        </Container>
      </Layout.Vertical>
      {packageType && (
        <>
          <Separator topSeparation="var(--spacing-large)" bottomSeparation="var(--spacing-large)" />
          <CreateRepositoryWidget packageType={packageType} type={type as RepositoryConfigType} />
        </>
      )}
    </Container>
  )
}

interface RepositoryCreateFormProps {
  factory?: RepositoryAbstractFactory
  defaultType?: RepositoryPackageType
  allowedPackageTypes?: RepositoryPackageType[]
  setShowOverlay: (show: boolean) => void
  onSuccess: (data: Repository) => void
}

function RepositoryCreateForm(props: RepositoryCreateFormProps, formikRef: FormikFowardRef): JSX.Element {
  const { defaultType, factory = repositoryFactory, onSuccess, setShowOverlay, allowedPackageTypes } = props
  const { getString } = useStrings()
  const parentRef = useGetSpaceRef()
  const { showSuccess, showError, clear } = useToaster()

  const { isLoading: createLoading, mutateAsync: createRepository } = useCreateRegistryMutation()

  useEffect(() => {
    setShowOverlay?.(createLoading)
  }, [createLoading])

  const getDefaultValuesByRepositoryType = useCallback(
    (repoType: RepositoryPackageType, defaultValue?: VirtualRegistryRequest): VirtualRegistryRequest => {
      const repositoryType = factory.getRepositoryType(repoType)
      if (repositoryType) {
        return repositoryType.getDefaultValues(defaultValue) as VirtualRegistryRequest
      }
      return {} as VirtualRegistryRequest
    },
    []
  )

  const getInitialValues = (): VirtualRegistryRequest => {
    const defaultSelectedPackageType = allowedPackageTypes?.length
      ? allowedPackageTypes[0]
      : RepositoryPackageType.DOCKER
    return getDefaultValuesByRepositoryType(defaultType ?? defaultSelectedPackageType)
  }

  const handleSubmit = async (values: VirtualRegistryRequest): Promise<void> => {
    try {
      const packageType = values?.packageType
      const repositoryType = factory.getRepositoryType(packageType)
      if (repositoryType) {
        const formattedValues = repositoryType.processRepositoryFormData(values) as VirtualRegistryRequest
        const formattedValuesForCleanupPolicy = getFormattedFormDataForCleanupPolicy(formattedValues)
        const response = await createRepository({
          queryParams: {
            space_ref: decodeRef(parentRef)
          },
          body: {
            ...formattedValuesForCleanupPolicy,
            parentRef: decodeRef(parentRef)
          } as RegistryRequestRequestBody
        })
        if (response.content.status === 'SUCCESS') {
          clear()
          showSuccess(getString('repositoryDetails.repositoryForm.repositoryCreated'))
          onSuccess(response.content.data as unknown as Repository)
        }
      }
    } catch (e: any) {
      showError(getErrorInfoFromErrorObject(e, true))
    }
  }

  return (
    <Formik<VirtualRegistryRequest>
      validationSchema={Yup.object().shape({
        identifier: Yup.string()
          .required(getString('validationMessages.nameRequired'))
          .trim()
          .matches(REPO_KEY_REGEX, getString('validationMessages.repokeyRegExMessage'))
      })}
      formName="repositoryForm"
      onSubmit={handleSubmit}
      initialValues={getInitialValues()}>
      {formik => {
        setFormikRef(formikRef, formik)
        return (
          <Container className={css.formContainer}>
            <FormContent
              allowedPackageTypes={allowedPackageTypes}
              formikProps={formik}
              getDefaultValuesByRepositoryType={getDefaultValuesByRepositoryType}
            />
          </Container>
        )
      }}
    </Formik>
  )
}

export default React.forwardRef(RepositoryCreateForm)
