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
import { Button, ButtonVariation, Container, Layout, Text } from '@harnessio/uicore'
import { Expander, Position } from '@blueprintjs/core'

import { useStrings } from '@ar/frameworks/strings'
import { PageType, RepositoryConfigType, type RepositoryPackageType } from '@ar/common/types'
import HeaderTitle from '@ar/components/Header/Title'
import { DEFAULT_DATE_TIME_FORMAT } from '@ar/constants'
import { getReadableDateTime } from '@ar/common/dateUtils'
import LabelsPopover from '@ar/components/LabelsPopover/LabelsPopover'
import RepositoryLocationBadge from '@ar/components/Badge/RepositoryLocationBadge'
import { useSetupClientModal } from '@ar/pages/repository-details/hooks/useSetupClientModal/useSetupClientModal'
import RepositoryIcon from '@ar/frameworks/RepositoryStep/RepositoryIcon'
import RepositoryActionsWidget from '@ar/frameworks/RepositoryStep/RepositoryActionsWidget'
import type { Repository } from '@ar/pages/repository-details/types'

interface RepositoryDetailsHeaderContentProps {
  data: Repository
  iconSize?: number
}

export default function RepositoryDetailsHeaderContent(props: RepositoryDetailsHeaderContentProps): JSX.Element {
  const { data, iconSize = 18 } = props
  const { identifier, labels, description, modifiedAt, packageType } = data || {}
  const { getString } = useStrings()
  const [showSetupClientModal] = useSetupClientModal({
    repoKey: identifier,
    packageType: packageType as RepositoryPackageType
  })
  return (
    <Container>
      <Layout.Horizontal spacing="medium" flex={{ alignItems: 'center' }}>
        <RepositoryIcon packageType={packageType as RepositoryPackageType} iconProps={{ size: iconSize }} />
        <Layout.Vertical spacing="small">
          <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
            <HeaderTitle>{identifier}</HeaderTitle>
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
          </Layout.Horizontal>
          <Text font={{ size: 'small' }} color={Color.GREY_500} width={800} lineClamp={1}>
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
        <Layout.Horizontal>
          <Button
            variation={ButtonVariation.PRIMARY}
            text={getString('actions.setupClient')}
            onClick={() => {
              showSetupClientModal()
            }}
            icon="setting"
          />
          <RepositoryActionsWidget
            type={RepositoryConfigType.VIRTUAL}
            packageType={data.packageType as RepositoryPackageType}
            data={data}
            readonly={false}
            pageType={PageType.Details}
          />
        </Layout.Horizontal>
      </Layout.Horizontal>
    </Container>
  )
}
