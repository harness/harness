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

import React from 'react'
import { FontVariation } from '@harnessio/design-system'
import { Card, Layout, Text } from '@harnessio/uicore'
import {
  listSpaceLabels,
  listSpaceLabelValues,
  TypesLabelValue,
  type TypesLabel
} from '@harnessio/react-har-service-client'

import { useGetSpaceRef } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'

import DropdownLabelSelector from './DropdownLabelSelector'

import css from './ManageMetadata.module.scss'

interface ManageMetadataLabelsProps {
  selectedLabels: any[] // update when we get type from BE
  onLabelSelect: (label: TypesLabel, value?: TypesLabelValue) => void
  disabled?: boolean
}

export default function ManageMetadataLabels(props: ManageMetadataLabelsProps) {
  const { getString } = useStrings()
  const spaceRef = useGetSpaceRef('')

  const getLabels = async (searchTerm: string): Promise<Array<TypesLabel>> => {
    return listSpaceLabels({
      space_ref: spaceRef,
      queryParams: {
        query: searchTerm,
        limit: 100,
        page: 1,
        inherited: false
      }
    }).then(response => response.content)
  }

  const getValues = async (label: TypesLabel, searchTerm: string): Promise<Array<TypesLabelValue>> => {
    return listSpaceLabelValues({
      space_ref: spaceRef,
      key: label.key ?? '',
      queryParams: {
        query: searchTerm,
        limit: 100,
        page: 1,
        inherited: false
      }
    }).then(response => response.content)
  }

  return (
    <Card className={css.container}>
      <Layout.Vertical flex={{ alignItems: 'flex-start' }} spacing="medium">
        <Text font={{ variation: FontVariation.BODY, weight: 'bold' }}>{getString('labels.title')}</Text>
        {/* TODO: render selected labels here... */}
        <DropdownLabelSelector getItems={getLabels} getValues={getValues} onSelect={props.onLabelSelect} />
      </Layout.Vertical>
    </Card>
  )
}
