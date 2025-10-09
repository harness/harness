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

import React, { useContext } from 'react'
import { FontVariation } from '@harnessio/design-system'
import { Card, Container, Layout, Text } from '@harnessio/uicore'

import { useParentHooks } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { VersionProviderContext } from '@ar/pages/version-details/context/VersionProvider'
import { LabelValueTypeEnum } from '@ar/pages/version-details/components/LabelValueContent/type'
import VersionDetailsTabs from '@ar/pages/version-details/components/VersionDetailsTabs/VersionDetailsTabs'
import { LabelValueContent } from '@ar/pages/version-details/components/LabelValueContent/LabelValueContent'

import type { DockerVersionDetailsQueryParams } from '../../types'

import css from './DockerVersionTreeNodeDetailsContent.module.scss'

export default function DockerVersionTreeNodeDetailsContent(): JSX.Element {
  const { useQueryParams } = useParentHooks()
  const { getString } = useStrings()
  const { data } = useContext(VersionProviderContext)
  const { digest } = useQueryParams<DockerVersionDetailsQueryParams>()
  if (!data) return <></>
  if (!digest) {
    return (
      <Layout.Vertical spacing="small" padding="large">
        <Text font={{ variation: FontVariation.CARD_TITLE }}>{getString('details')}</Text>
        <Card className={css.cardContainer}>
          <Container className={css.gridContainer}>
            <LabelValueContent
              label={getString('versionDetails.overview.generalInformation.name')}
              value={data.imageName}
              type={LabelValueTypeEnum.Text}
            />
            <LabelValueContent
              label={getString('versionDetails.overview.generalInformation.version')}
              value={data.version}
              type={LabelValueTypeEnum.Text}
            />
            <LabelValueContent
              label={getString('versionDetails.overview.generalInformation.packageType')}
              value={getString('packageTypes.dockerPackage')}
              type={LabelValueTypeEnum.PackageType}
              icon="docker-step"
            />
          </Container>
        </Card>
      </Layout.Vertical>
    )
  }
  return <VersionDetailsTabs />
}
