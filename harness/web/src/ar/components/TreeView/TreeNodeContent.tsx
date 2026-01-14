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
import { Layout, Text } from '@harnessio/uicore'
import { Icon, type IconName } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'

import { useStrings } from '@ar/frameworks/strings'
import { RepositoryConfigType } from '@ar/common/types'

import css from './TreeView.module.scss'

interface TreeNodeContentProps {
  icon?: IconName
  iconSize?: number
  label: string
  downloads?: number
  size?: string
  artifacts?: number
  type?: RepositoryConfigType
  compact?: boolean
}

export default function TreeNodeContent(props: TreeNodeContentProps) {
  const { icon, iconSize, label, type, downloads, artifacts, size, compact } = props
  const { getString } = useStrings()
  return (
    <Layout.Horizontal className={css.treeNodeContent} flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
      {icon && <Icon name={icon} size={iconSize} />}
      <Layout.Vertical className={css.labelContainer} spacing="xsmall">
        <Text font={{ variation: FontVariation.BODY }} lineClamp={1}>
          {label}
        </Text>
        {!compact && (
          <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'flex-start' }} spacing="xsmall">
            {type && (
              <>
                <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_400} lineClamp={1}>
                  {type === RepositoryConfigType.VIRTUAL
                    ? getString('badges.artifactRegistry')
                    : getString('badges.upstreamProxy')}
                </Text>
              </>
            )}

            {size !== undefined && (
              <Text
                icon="dot"
                iconProps={{ size: 8, color: Color.GREY_400 }}
                font={{ variation: FontVariation.SMALL }}
                color={Color.GREY_400}
                lineClamp={1}>
                {size}
              </Text>
            )}
            {artifacts !== undefined && (
              <Text
                icon="dot"
                iconProps={{ size: 8, color: Color.GREY_400 }}
                rightIcon="store-artifact-bundle"
                rightIconProps={{ size: 12 }}
                color={Color.GREY_400}
                font={{ variation: FontVariation.SMALL }}
                lineClamp={1}>
                {artifacts}
              </Text>
            )}
            {downloads !== undefined && (
              <Text
                icon="dot"
                iconProps={{ size: 8, color: Color.GREY_400 }}
                rightIcon="download-box"
                rightIconProps={{ size: 12 }}
                color={Color.GREY_400}
                font={{ variation: FontVariation.SMALL }}
                lineClamp={1}>
                {downloads.toLocaleString()}
              </Text>
            )}
          </Layout.Horizontal>
        )}
      </Layout.Vertical>
    </Layout.Horizontal>
  )
}
