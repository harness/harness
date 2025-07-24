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
import { Spinner } from '@blueprintjs/core'
import { getErrorInfoFromErrorObject, Layout, PageError, Text } from '@harnessio/uicore'

import type { ColorName } from 'utils/Utils'
import { type TypesLabel, useListSpaceLabelValues } from 'services/code'

import { useGetSpaceRef } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import LabelValuesListContent from './LabelValuesListContent'

interface LabelValuesListProps {
  data: TypesLabel
}

function LabelValuesList({ data }: LabelValuesListProps) {
  const spaceRef = useGetSpaceRef()
  const { getString } = useStrings()
  const {
    data: response,
    loading,
    error,
    refetch
  } = useListSpaceLabelValues({
    space_ref: spaceRef,
    key: data.key || '',
    lazy: !data.key
  })
  if (loading) return <Spinner size={Spinner.SIZE_SMALL} />
  if (error) return <PageError message={getErrorInfoFromErrorObject(error)} onClick={() => refetch()} />
  if (!response || !response.length) return <Text>{getString('labelsList.table.noData')}</Text>
  return (
    <Layout.Horizontal spacing="xlarge">
      <LabelValuesListContent
        list={response}
        labelName={data.key}
        scope={data.scope}
        color={data.color as ColorName}
        allowDynamicValues={data.type === 'dynamic'}
      />
    </Layout.Horizontal>
  )
}

export default LabelValuesList
