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

import React from 'react'
import cx from 'classnames'
import { Icon } from '@harnessio/icons'
import { Container, Layout, Text, Link } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'
import { useAppContext } from 'AppContext'
import { ThreadSection } from 'components/ThreadSection/ThreadSection'
import { FeatureType, type FeatureData } from 'utils/Utils'
import { useStrings } from 'framework/strings'
import featuresData from './featureRoadmap.json'
import featuresStandaloneData from './featureRoadmapStandalone.json'

import Launch from '../../../icons/Launch.svg?url'

import css from '../RepositoriesListing.module.scss'

const FeatureMap = () => {
  const { getString } = useStrings()
  const { standalone } = useAppContext()

  const features = standalone ? featuresStandaloneData : featuresData
  return (
    <Container padding={'medium'} className={css.featureContainer} width={`285px`}>
      <Text className={css.featureText} color={Color.GREY_400} padding={{ top: 'large', bottom: 'small' }}>
        {getString('featureRoadmap')}
      </Text>
      {features.map((feature: FeatureData) => {
        const typeText = (feature.typeText as string).toLocaleUpperCase()
        return (
          <ThreadSection
            key={`${feature.title}`}
            title={
              <Layout.Horizontal
                className={cx(
                  { [css.comingSoonContainer]: feature.type === FeatureType.COMINGSOON },
                  { [css.releasedContainer]: feature.type === FeatureType.RELEASED }
                )}>
                <Icon className={css.iconContainer} name="dot" size={16} />
                <Container padding="xsmall">
                  <Container className={css.tagBackground} padding="xsmall">
                    <Text className={cx(css.tagText)}>{typeText}</Text>
                  </Container>
                </Container>
              </Layout.Horizontal>
            }>
            {
              <Container>
                <Layout.Vertical>
                  <Text className={css.featureTitle}>
                    {feature.title}
                    <Link target={'_blank'} rel={'noopener noreferrer'} noStyling href={feature.link}>
                      <Container padding={{ top: 'tiny', left: 'small' }}>
                        <img className={css.launchIcon} src={Launch} width={12} height={12}></img>
                      </Container>
                    </Link>
                  </Text>
                  <Text className={css.featureContent}>{feature.content}</Text>
                </Layout.Vertical>
              </Container>
            }
          </ThreadSection>
        )
      })}
    </Container>
  )
}

export default FeatureMap
