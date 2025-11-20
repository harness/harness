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

import React, { forwardRef } from 'react'
import { Card, getErrorInfoFromErrorObject, useToaster } from '@harnessio/uicore'
import { useGetMetadataQuery, useUpdateMetadataMutation } from '@harnessio/react-har-service-v2-client'

import { useAppStore } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { queryClient } from '@ar/utils/queryClient'
import type { FormikFowardRef } from '@ar/common/types'
import PageContent from '@ar/components/PageContent/PageContent'
import PropertiesForm from '@ar/components/PropertiesForm/PropertiesForm'
import type { PropertiesFormValues } from './types'

interface PropertiesFormContentProps {
  repositoryIdentifier: string
  setIsDirty: (dirty: boolean) => void
  setIsUpdating: (updating: boolean) => void
  className?: string
  readonly?: boolean
  artifactIdentifier?: string
  versionIdentifier?: string
}

function PropertiesFormContent(props: PropertiesFormContentProps, formikRef: FormikFowardRef) {
  const {
    readonly,
    repositoryIdentifier,
    setIsDirty,
    setIsUpdating,
    className,
    artifactIdentifier,
    versionIdentifier
  } = props
  const { scope } = useAppStore()
  const { showSuccess, showError, clear } = useToaster()
  const { getString } = useStrings()

  const { data, isFetching, error, refetch } = useGetMetadataQuery({
    queryParams: {
      account_identifier: scope.accountId as string,
      registry_identifier: repositoryIdentifier,
      package: artifactIdentifier,
      version: versionIdentifier
    }
  })
  const initialValue = data?.content?.data?.metadata || []

  const { mutateAsync: updateMetadata } = useUpdateMetadataMutation()

  const handleUpdateMetadata = async (values: PropertiesFormValues) => {
    try {
      setIsUpdating(true)
      const response = await updateMetadata({
        body: {
          metadata: values.value,
          registryIdentifier: repositoryIdentifier,
          package: artifactIdentifier,
          version: versionIdentifier
        },
        queryParams: {
          account_identifier: scope.accountId as string
        }
      })
      if (response.content.status === 'SUCCESS') {
        clear()
        showSuccess(getString('repositoryDetails.repositoryForm.repositoryUpdated'))
        queryClient.invalidateQueries(['GetMetadata'])
      }
    } catch (e: any) {
      showError(getErrorInfoFromErrorObject(e, true))
    } finally {
      setIsUpdating(false)
    }
  }

  return (
    <PageContent loading={isFetching} error={error} refetch={refetch}>
      <Card className={className}>
        <PropertiesForm
          readonly={readonly}
          value={{ value: initialValue }}
          ref={formikRef}
          onChangeDirty={setIsDirty}
          onSubmit={handleUpdateMetadata}
        />
      </Card>
    </PageContent>
  )
}

PropertiesFormContent.displayName = 'PropertiesFormContent'

export default forwardRef(PropertiesFormContent)
