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
import { Color } from '@harnessio/design-system'
import { Container, Layout, Text } from '@harnessio/uicore'
import { Expander, Position } from '@blueprintjs/core'

import { useAppStore } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { PageType, RepositoryConfigType, RepositoryVisibility, type RepositoryPackageType } from '@ar/common/types'
import HeaderTitle from '@ar/components/Header/Title'
import { DEFAULT_DATE_TIME_FORMAT } from '@ar/constants'
import { getReadableDateTime } from '@ar/common/dateUtils'
import LabelsPopover from '@ar/components/LabelsPopover/LabelsPopover'
import SetupClientButton from '@ar/components/SetupClientButton/SetupClientButton'
import RepositoryLocationBadge from '@ar/components/Badge/RepositoryLocationBadge'
import RepositoryIcon from '@ar/frameworks/RepositoryStep/RepositoryIcon'
import RepositoryActionsWidget from '@ar/frameworks/RepositoryStep/RepositoryActionsWidget'
import type { Repository } from '@ar/pages/repository-details/types'
import RepositoryVisibilityBadge from '@ar/components/Badge/RepositoryVisibilityBadge'
import AvailablityBadge, { AvailablityBadgeType } from '@ar/components/Badge/AvailablityBadge'

import css from './RepositoryDetailsHeader.module.scss'

interface RepositoryDetailsHeaderContentProps {
  data: Repository
  iconSize?: number
}

export default function RepositoryDetailsHeaderContent(props: RepositoryDetailsHeaderContentProps): JSX.Element {
  const { data, iconSize = 40 } = props
  const { identifier, labels, description, modifiedAt, packageType, isPublic, isDeleted } = data || {}
  const { isCurrentSessionPublic } = useAppStore()
  const { getString } = useStrings()
  return (
    <Container>
      <Layout.Horizontal data-testid="registry-header-container" spacing="medium" flex={{ alignItems: 'center' }}>
        <RepositoryIcon packageType={packageType as RepositoryPackageType} iconProps={{ size: iconSize }} />
        <Layout.Vertical spacing="small" className={css.nameContainer}>
          <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
            <HeaderTitle data-testid="registry-title">{identifier}</HeaderTitle>
            <RepositoryLocationBadge type={RepositoryConfigType.VIRTUAL} />
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
            <AvailablityBadge type={isDeleted ? AvailablityBadgeType.ARCHIVED : AvailablityBadgeType.AVAILABLE} />
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
            <Text data-testid="registry-last-modified-at" font={{ size: 'small' }}>
              {modifiedAt ? getReadableDateTime(Number(modifiedAt), DEFAULT_DATE_TIME_FORMAT) : getString('na')}
            </Text>
          </Layout.Horizontal>
        </Layout.Vertical>
        <Expander />
        <Layout.Horizontal>
          <SetupClientButton repositoryIdentifier={identifier} packageType={packageType as RepositoryPackageType} />
          {!isCurrentSessionPublic && (
            <RepositoryActionsWidget
              type={RepositoryConfigType.VIRTUAL}
              packageType={data.packageType as RepositoryPackageType}
              data={data}
              readonly={false}
              pageType={PageType.Details}
            />
          )}
        </Layout.Horizontal>
      </Layout.Horizontal>
    </Container>
  )
}
