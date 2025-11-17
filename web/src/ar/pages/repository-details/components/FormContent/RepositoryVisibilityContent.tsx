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
import { useFormikContext } from 'formik'
import { Color, FontVariation } from '@harnessio/design-system'
import { CardSelect, CardSelectType, Layout, Text } from '@harnessio/uicore'

import { useAppStore } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { RepositoryVisibility } from '@ar/common/types'
import CollapseContainer from '@ar/components/CollapseContainer/CollapseContainer'
import type { UpstreamRegistryRequest } from '@ar/pages/upstream-proxy-details/types'

import { RepositoryVisibilityOptions } from '../../constants'
import type { VirtualRegistryRequest } from '../../types'
import css from './FormContent.module.scss'

interface RepositoryVisibilityContentProps {
  disabled: boolean
}

export default function RepositoryVisibilityContent(props: RepositoryVisibilityContentProps) {
  const { disabled } = props
  const { values, setFieldValue } = useFormikContext<VirtualRegistryRequest | UpstreamRegistryRequest>()
  const { isPublicAccessEnabledOnResources } = useAppStore()
  const { getString } = useStrings()
  const isPublic = !!values.isPublic
  if (!isPublicAccessEnabledOnResources) return null
  return (
    <CollapseContainer
      titleClassName={css.visibilityTitle}
      title={getString('repositoryDetails.repositoryForm.visibility.title')}
      initialState={true}>
      <CardSelect
        className={css.visibilityCardSelect}
        selected={isPublic ? RepositoryVisibility.PUBLIC : RepositoryVisibility.PRIVATE}
        data={[RepositoryVisibility.PUBLIC, RepositoryVisibility.PRIVATE]}
        multi={false}
        type={CardSelectType.CardView}
        cornerSelected={true}
        onChange={selected => {
          if (disabled) return
          setFieldValue('isPublic', selected === RepositoryVisibility.PUBLIC)
        }}
        renderItem={item => {
          const option = RepositoryVisibilityOptions[item]
          if (!option) return <></>
          return (
            <Layout.Vertical>
              <Text font={{ variation: FontVariation.BODY, weight: 'bold' }}>{getString(option.label)}</Text>
              <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_600}>
                {getString(option.description)}
              </Text>
            </Layout.Vertical>
          )
        }}></CardSelect>
    </CollapseContainer>
  )
}
