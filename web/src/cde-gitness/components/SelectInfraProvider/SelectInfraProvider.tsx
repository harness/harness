/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useEffect, useState } from 'react'
import { groupBy } from 'lodash-es'
import { Layout } from '@harnessio/uicore'
import { useFormikContext } from 'formik'
import { useParams } from 'react-router-dom'
import type { OpenapiCreateGitspaceRequest, TypesInfraProviderConfig, TypesInfraProviderResource } from 'services/cde'
import { useInfraListingApi } from 'cde-gitness/hooks/useGetInfraListProvider'
import { HARNESS_GCP, HYBRID_VM_GCP, type dropdownProps } from 'cde-gitness/constants'
import { useAppContext } from 'AppContext'
import { SelectRegion } from '../SelectRegion/SelectRegion'
import { SelectMachine } from '../SelectMachine/SelectMachine'
import SelectInfraProviderType from '../SelectInfraProviderType/SelectInfraProviderType'

export const SelectInfraProvider = () => {
  const { hooks } = useAppContext()
  const { CDE_HYBRID_ENABLED, CDE_HARNESS_GCP_ENABLED } = hooks?.useFeatureFlags()
  const [infraProviders, setInfraProvider] = useState<dropdownProps[]>()
  const { values, setFieldValue: onChange } = useFormikContext<OpenapiCreateGitspaceRequest>()

  const { data, refetch } = useInfraListingApi({
    queryParams: {
      acl_filter: 'true'
    }
  })

  useEffect(() => {
    refetch()
  }, [])

  const { gitspaceId = '' } = useParams<{ gitspaceId?: string }>()

  const [optionsList, setOptionList] = useState<TypesInfraProviderResource[]>([])

  useEffect(() => {
    const infraOptions: dropdownProps[] = []
    data?.forEach(infra => {
      const payload = {
        label: infra?.name ?? '',
        value: infra?.identifier ?? ''
      }
      if (infra.type === HARNESS_GCP && CDE_HARNESS_GCP_ENABLED) {
        infraOptions.push(payload)
      }
      if (infra.type === HYBRID_VM_GCP && CDE_HYBRID_ENABLED) {
        infraOptions.push(payload)
      }
      if (infra.type !== HARNESS_GCP && infra.type !== HYBRID_VM_GCP) {
        infraOptions.push(payload)
      }
    })
    setInfraProvider(infraOptions)
  }, [data])

  useEffect(() => {
    let options: TypesInfraProviderResource[] = []
    if (values?.metadata?.infraProvider) {
      data?.forEach((infra: TypesInfraProviderConfig) => {
        if (infra?.identifier === values?.metadata?.infraProvider) {
          options = infra?.resources ?? []
        }
      })
    }
    setOptionList(options)
  }, [values?.metadata?.infraProvider])

  useEffect(() => {
    if (gitspaceId && values.resource_identifier && optionsList?.length) {
      const match = optionsList.find(item => item.identifier === values.resource_identifier)
      if (values?.metadata?.region !== match?.region) {
        onChange('metadata.region', match?.region?.toLowerCase())
      }
    }
  }, [gitspaceId, values.resource_identifier, values?.metadata?.region, optionsList.map(i => i.name).join('')])

  const regionOptions = Object.entries(groupBy(optionsList, 'region')).map(i => {
    return { label: i[0] as string, value: i[1] }
  })

  const machineOptions =
    optionsList
      ?.filter(item => item?.region?.toLowerCase() === values?.metadata?.region?.toLowerCase())
      ?.map(item => {
        return { ...item }
      }) || []

  return (
    <Layout.Vertical spacing="small">
      <SelectInfraProviderType infraProviders={infraProviders ?? []} allProviders={data ?? []} />
      <SelectRegion
        defaultValue={regionOptions?.[0]}
        options={regionOptions}
        isDisabled={regionOptions?.length === 0}
      />
      <SelectMachine
        options={machineOptions}
        defaultValue={machineOptions?.[0]}
        isDisabled={machineOptions?.length === 0}
      />
    </Layout.Vertical>
  )
}
