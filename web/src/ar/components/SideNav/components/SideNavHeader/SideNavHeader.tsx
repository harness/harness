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
import classNames from 'classnames'
import { Link } from 'react-router-dom'
import { Icon, IconName } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { Container, Layout, Text } from '@harnessio/uicore'

import { useRoutes } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings/String'
import type { StringKeys } from '@ar/frameworks/strings'

import css from './SideNavHeader.module.scss'

export interface ModuleInfo {
  icon: IconName
  label: StringKeys
  color: string
  backgroundColor?: string
  backgroundColorLight?: string
  isNew?: boolean
  isModuleLicensed?: boolean
}

const moduleConfig: ModuleInfo = {
  icon: 'artifact-registry',
  label: 'harLabel',
  color: '--har-border',
  isNew: true,
  isModuleLicensed: false
}

export default function SideNavHeader(): JSX.Element {
  const { color, icon, isNew, label } = moduleConfig
  const { getString } = useStrings()
  const routes = useRoutes()

  return (
    <Container padding={{ left: 'medium', right: 'medium', bottom: 'medium' }} className={css.width100}>
      <Layout.Horizontal
        flex={{ justifyContent: 'space-between' }}
        className={classNames(css.container)}
        style={{ borderColor: color ? `var(${color})` : 'var(--primary-6)' }}>
        <Link className={css.link} to={routes.toAR()}>
          <Layout.Horizontal flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
            <Icon
              className={classNames({ [css.harnessLogo]: !icon })}
              name={icon || 'harness-logo-white'}
              size={icon ? 32 : 110}
              margin={{ right: 'small' }}
            />

            {isNew && (
              <Text
                font={{ variation: FontVariation.TINY_SEMI }}
                padding="xxsmall"
                color={Color.PURPLE_800}
                background={Color.PURPLE_50}
                className={css.newTag}>
                {getString('new').toUpperCase()}
              </Text>
            )}

            {label && (
              <Text color={Color.WHITE} font={{ variation: FontVariation.BODY2 }} className={css.label}>
                {getString(label)}
              </Text>
            )}
          </Layout.Horizontal>
        </Link>
      </Layout.Horizontal>
    </Container>
  )
}
