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

import React, { Fragment } from 'react'
import cx from 'classnames'
import { Container, Layout, Text, PageHeader, PageHeaderProps } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { Link, useParams } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import type { CODEProps } from 'RouteDefinitions'
import type { GitInfoProps } from 'utils/GitUtils'
import css from './RepositoryPageHeader.module.scss'

interface BreadcrumbLink {
  label: string
  url: string
}

interface RepositoryPageHeaderProps extends Optional<Pick<GitInfoProps, 'repoMetadata'>, 'repoMetadata'> {
  title: string | JSX.Element
  dataTooltipId: string
  extraBreadcrumbLinks?: BreadcrumbLink[]
  className?: string
  content?: PageHeaderProps['content']
}

export function RepositoryPageHeader({
  repoMetadata,
  title,
  dataTooltipId,
  extraBreadcrumbLinks = [],
  className,
  content
}: RepositoryPageHeaderProps) {
  const { gitRef } = useParams<CODEProps>()
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const { routes, isCurrentSessionPublic } = useAppContext()

  return (
    <PageHeader
      className={className}
      content={content}
      title=""
      breadcrumbs={
        <Container className={css.header}>
          <Layout.Horizontal
            spacing="small"
            className={cx(css.breadcrumb, { [css.hideBreadcrumbs]: isCurrentSessionPublic })}>
            <Link to={routes.toCODERepositories({ space })}>{getString('repositories')}</Link>
            <Icon name="main-chevron-right" size={8} color={Color.GREY_500} />
            <Link to={routes.toCODERepository({ repoPath: (repoMetadata?.path as string) || '', gitRef })}>
              {repoMetadata?.uid || ''}
            </Link>
            {extraBreadcrumbLinks.map(link => (
              <Fragment key={link.url}>
                <Icon name="main-chevron-right" size={8} color={Color.GREY_500} />
                {/* This allows for outer most entities to not necessarily be links */}
                {link.url ? (
                  <Link to={link.url}>{link.label}</Link>
                ) : (
                  <Text font={{ variation: FontVariation.SMALL }} color={Color.GREY_500}>
                    {link.label}
                  </Text>
                )}
              </Fragment>
            ))}
          </Layout.Horizontal>
          <Container padding={{ top: 'small', bottom: 'small' }}>
            {typeof title === 'string' ? (
              <Text tag="h1" font={{ variation: FontVariation.H4 }} tooltipProps={{ dataTooltipId }}>
                {title}
              </Text>
            ) : (
              title
            )}
          </Container>
        </Container>
      }
    />
  )
}
