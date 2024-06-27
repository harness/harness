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

import React, { useEffect } from 'react'
import { isObject, groupBy } from 'lodash-es'
import { Layout } from '@harnessio/uicore'
import { useFormikContext } from 'formik'
import { useParams } from 'react-router-dom'
import { CDEPathParams, useGetCDEAPIParams } from 'cde/hooks/useGetCDEAPIParams'
import { OpenapiCreateGitspaceRequest, useListInfraProviderResourcesForAccount } from 'services/cde'
import { SelectRegion } from '../SelectRegion/SelectRegion'
import { SelectMachine } from '../SelectMachine/SelectMachine'

export const SelectInfraProvider = () => {
  const { values, setFieldValue: onChange } = useFormikContext<OpenapiCreateGitspaceRequest>()
  const { accountIdentifier } = useGetCDEAPIParams() as CDEPathParams
  const { data } = useListInfraProviderResourcesForAccount({
    accountIdentifier,
    infraProviderConfigIdentifier: 'HARNESS_GCP'
  })

  const { gitspaceId = '' } = useParams<{ gitspaceId?: string }>()

  const optionsList = data && isObject(data) ? data : []

  useEffect(() => {
    if (gitspaceId && values.infra_provider_resource_id && optionsList.length) {
      const match = optionsList.find(item => item.id === values.infra_provider_resource_id)
      if (values?.metadata?.region !== match?.region) {
        onChange('metadata.region', match?.region?.toLowerCase())
      }
    }
  }, [gitspaceId, values.infra_provider_resource_id, values?.metadata?.region, optionsList.map(i => i.name).join('')])

  const regionOptions = Object.entries(groupBy(optionsList, 'region')).map(i => {
    return { label: i[0], value: i[1] }
  })

  const machineOptions =
    optionsList
      ?.filter(item => item?.region === values?.metadata?.region)
      ?.map(item => {
        return { ...item }
      }) || []

  return (
    <Layout.Horizontal spacing="medium">
      <SelectRegion defaultValue={regionOptions?.[0]} options={regionOptions} disabled={!!gitspaceId} />
      <SelectMachine options={machineOptions} defaultValue={machineOptions?.[0]} />
    </Layout.Horizontal>
  )
}
