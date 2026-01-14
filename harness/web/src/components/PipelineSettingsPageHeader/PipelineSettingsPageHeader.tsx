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
import { Layout, PageHeader, Container } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color } from '@harnessio/design-system'
import { Link, useParams } from 'react-router-dom'
import { useStrings } from 'framework/strings'
import { useAppContext } from 'AppContext'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import type { CODEProps } from 'RouteDefinitions'
import type { GitInfoProps } from 'utils/GitUtils'
import { TabOptions } from 'components/PipelineSettings/PipelineSettings'
import css from './PipelineSettingsPageHeader.module.scss'

interface BreadcrumbLink {
  label: string
  url: string
}

interface PipelineSettingsPageHeaderProps extends Optional<Pick<GitInfoProps, 'repoMetadata'>, 'repoMetadata'> {
  title: string | JSX.Element
  dataTooltipId: string
  extraBreadcrumbLinks?: BreadcrumbLink[]
  selectedTab: TabOptions
  setSelectedTab: (tab: TabOptions) => void
}

const PipelineSettingsPageHeader = ({
  repoMetadata,
  title,
  extraBreadcrumbLinks = [],
  selectedTab,
  setSelectedTab
}: PipelineSettingsPageHeaderProps) => {
  const { gitRef } = useParams<CODEProps>()
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const { routes } = useAppContext()

  if (!repoMetadata) {
    return null
  }

  return (
    <PageHeader
      className={css.pageHeader}
      title={title}
      breadcrumbs={
        <Layout.Horizontal
          spacing="small"
          className={css.breadcrumb}
          padding={{ bottom: 0 }}
          margin={{ bottom: 'small' }}>
          <Link to={routes.toCODERepositories({ space })}>{getString('repositories')}</Link>
          <Icon name="main-chevron-right" size={8} color={Color.GREY_500} />
          <Link to={routes.toCODERepository({ repoPath: repoMetadata.path as string, gitRef })}>
            {repoMetadata.identifier}
          </Link>
          {extraBreadcrumbLinks.map(link => (
            <Fragment key={link.url}>
              <Icon name="main-chevron-right" size={8} color={Color.GREY_500} />
              <Link to={link.url}>{link.label}</Link>
            </Fragment>
          ))}
        </Layout.Horizontal>
      }
      content={
        <Container className={css.tabs}>
          {Object.values(TabOptions).map(tabOption => (
            <div
              key={tabOption}
              className={`${css.tab} ${selectedTab === tabOption ? css.active : ''}`}
              onClick={() => setSelectedTab(tabOption)}>
              {tabOption}
            </div>
          ))}
        </Container>
      }
    />
  )
}

export default PipelineSettingsPageHeader
