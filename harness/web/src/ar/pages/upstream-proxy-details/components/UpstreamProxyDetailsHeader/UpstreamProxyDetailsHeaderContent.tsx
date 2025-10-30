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
import { defaultTo, isEmpty } from 'lodash-es'
import { Expander, Position } from '@blueprintjs/core'
import { Color } from '@harnessio/design-system'
import { Container, Layout, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { DEFAULT_DATE_TIME_FORMAT } from '@ar/constants'
import HeaderTitle from '@ar/components/Header/Title'
import { getReadableDateTime } from '@ar/common/dateUtils'
import type { Repository } from '@ar/pages/repository-details/types'
import LabelsPopover from '@ar/components/LabelsPopover/LabelsPopover'
import RepositoryIcon from '@ar/frameworks/RepositoryStep/RepositoryIcon'
import RepositoryLocationBadge from '@ar/components/Badge/RepositoryLocationBadge'
import RepositoryVisibilityBadge from '@ar/components/Badge/RepositoryVisibilityBadge'
import RepositoryActionsWidget from '@ar/frameworks/RepositoryStep/RepositoryActionsWidget'
import { PageType, RepositoryConfigType, RepositoryPackageType, RepositoryVisibility } from '@ar/common/types'

import css from './UpstreamProxyDetailsHeader.module.scss'

interface UpstreamProxyDetailsHeaderContentProps {
  data: Repository
  iconSize?: number
}

export default function UpstreamProxyDetailsHeaderContent(props: UpstreamProxyDetailsHeaderContentProps): JSX.Element {
  const { data, iconSize = 40 } = props
  const { identifier, modifiedAt, packageType, labels, description, isPublic } = data
  const { getString } = useStrings()
  return (
    <Container>
      <Layout.Horizontal
        data-testid="upstream-registry-header-container"
        spacing="medium"
        flex={{ alignItems: 'center' }}>
        <RepositoryIcon packageType={packageType as RepositoryPackageType} iconProps={{ size: iconSize }} />
        <Layout.Vertical spacing="small" className={css.nameContainer}>
          <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
            <HeaderTitle>{identifier}</HeaderTitle>
            <RepositoryLocationBadge type={RepositoryConfigType.UPSTREAM} />
            {!isEmpty(labels) && (
              <LabelsPopover
                withCount
                iconProps={{ size: 14, color: Color.GREY_600 }}
                labels={defaultTo(labels, [])}
                popoverProps={{
                  position: Position.RIGHT
                }}
                tagsTitle={getString('tagsLabel')}
              />
            )}
            <RepositoryVisibilityBadge type={isPublic ? RepositoryVisibility.PUBLIC : RepositoryVisibility.PRIVATE} />
          </Layout.Horizontal>
          <Text
            data-testid="registry-description"
            font={{ size: 'small' }}
            color={Color.GREY_500}
            width={800}
            lineClamp={1}>
            {description || getString('noDescription')}
          </Text>
          <Layout.Horizontal spacing="small">
            <Text font={{ size: 'small', weight: 'semi-bold' }} color={Color.BLACK} margin={{ right: 'small' }}>
              {getString('lastUpdated')}:
            </Text>
            <Text font={{ size: 'small' }}>
              {modifiedAt ? getReadableDateTime(Number(modifiedAt), DEFAULT_DATE_TIME_FORMAT) : getString('na')}
            </Text>
          </Layout.Horizontal>
        </Layout.Vertical>
        <Expander />
        <RepositoryActionsWidget
          type={RepositoryConfigType.UPSTREAM}
          packageType={data.packageType as RepositoryPackageType}
          data={data}
          readonly={false}
          pageType={PageType.Details}
        />
      </Layout.Horizontal>
    </Container>
  )
}
