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

import React, { useCallback, useContext, useMemo, useState } from 'react'
import { Redirect, Switch, useHistory } from 'react-router-dom'
import { Container, Tab, Tabs } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { useQueryParams } from '@ar/__mocks__/hooks'
import { DEFAULT_ORG, DEFAULT_PROJECT } from '@ar/constants'
import { useAppStore, useDecodedParams, useRoutes } from '@ar/hooks'
import type { RepositoryPackageType } from '@ar/common/types'
import type { VersionDetailsPathParams } from '@ar/routes/types'
import versionFactory from '@ar/frameworks/Version/VersionFactory'
import RouteProvider from '@ar/components/RouteProvider/RouteProvider'
import VersionDetailsTabWidget from '@ar/frameworks/Version/VersionDetailsTabWidget'
import { VersionProviderContext } from '@ar/pages/version-details/context/VersionProvider'
import {
  versionDetailsPathParams,
  versionDetailsTabPathParams,
  versionDetailsTabWithOrgAndProjectPathParams,
  versionDetailsTabWithPipelineDetailsPathParams,
  versionDetailsTabWithProjectPathParams,
  versionDetailsTabWithSSCADetailsPathParams
} from '@ar/routes/RouteDestinations'

import { VersionDetailsTab, VersionDetailsTabList } from './constants'
import type { DockerVersionDetailsQueryParams } from '../../DockerVersion/types'
import css from './VersionDetailsTab.module.scss'

export default function VersionDetailsTabs(): JSX.Element {
  const [tab, setTab] = useState('')

  const routes = useRoutes()
  const { scope } = useAppStore()
  const history = useHistory()
  const { getString } = useStrings()
  const routeDefinitions = useRoutes(true)
  const { data } = useContext(VersionProviderContext)
  const pathParams = useDecodedParams<VersionDetailsPathParams>()
  const { digest } = useQueryParams<DockerVersionDetailsQueryParams>()
  const { orgIdentifier, projectIdentifier } = scope

  const tabList = useMemo(() => {
    const versionType = versionFactory?.getVersionType(data?.packageType)
    if (!versionType) return []
    return VersionDetailsTabList.filter(each => versionType.getAllowedVersionDetailsTab().includes(each.value))
  }, [data])

  const handleTabChange = useCallback(
    (nextTab: VersionDetailsTab): void => {
      setTab(nextTab)
      let newRoute
      switch (nextTab) {
        case VersionDetailsTab.SUPPLY_CHAIN:
          newRoute = routes.toARVersionDetailsTab({
            versionIdentifier: pathParams.versionIdentifier,
            artifactIdentifier: pathParams.artifactIdentifier,
            repositoryIdentifier: pathParams.repositoryIdentifier,
            versionTab: nextTab,
            sourceId: data?.sscaArtifactSourceId,
            artifactId: data?.sscaArtifactId,
            orgIdentifier: !orgIdentifier ? DEFAULT_ORG : undefined,
            projectIdentifier: !projectIdentifier ? DEFAULT_PROJECT : undefined
          })
          break
        case VersionDetailsTab.SECURITY_TESTS:
          newRoute = routes.toARVersionDetailsTab({
            versionIdentifier: pathParams.versionIdentifier,
            artifactIdentifier: pathParams.artifactIdentifier,
            repositoryIdentifier: pathParams.repositoryIdentifier,
            versionTab: nextTab,
            executionIdentifier: data?.stoExecutionId,
            pipelineIdentifier: data?.stoPipelineId,
            orgIdentifier: !orgIdentifier ? DEFAULT_ORG : undefined,
            projectIdentifier: !projectIdentifier ? DEFAULT_PROJECT : undefined
          })
          break
        default:
          newRoute = routes.toARVersionDetailsTab({
            versionIdentifier: pathParams.versionIdentifier,
            artifactIdentifier: pathParams.artifactIdentifier,
            repositoryIdentifier: pathParams.repositoryIdentifier,
            versionTab: nextTab
          })
          break
      }
      if (digest) {
        newRoute = `${newRoute}?digest=${digest}`
      }
      history.push(newRoute)
    },
    [digest]
  )

  if (!data) return <></>
  return (
    <>
      <Container className={css.tabsContainer}>
        <Tabs id="versionDetailsTab" selectedTabId={tab} onChange={handleTabChange}>
          {tabList.map(each => (
            <Tab key={each.value} id={each.value} disabled={each.disabled} title={getString(each.label)} />
          ))}
        </Tabs>
      </Container>
      <Switch>
        <RouteProvider exact path={routeDefinitions.toARVersionDetails({ ...versionDetailsPathParams })}>
          <Redirect
            to={routes.toARVersionDetailsTab({
              versionIdentifier: pathParams.versionIdentifier,
              artifactIdentifier: pathParams.artifactIdentifier,
              repositoryIdentifier: pathParams.repositoryIdentifier,
              versionTab: VersionDetailsTab.OVERVIEW
            })}
          />
        </RouteProvider>
        <RouteProvider
          exact
          path={[
            routeDefinitions.toARVersionDetailsTab({ ...versionDetailsTabPathParams }),
            // with project and org data
            routeDefinitions.toARVersionDetailsTab({
              ...versionDetailsTabWithOrgAndProjectPathParams
            }),
            // ssca with pipeline data
            routeDefinitions.toARVersionDetailsTab({ ...versionDetailsTabWithSSCADetailsPathParams }),
            // ssca with project and pipeline data
            routeDefinitions.toARVersionDetailsTab({
              ...versionDetailsTabWithSSCADetailsPathParams,
              ...versionDetailsTabWithProjectPathParams
            }),
            // ssca with org, project and pipeline data
            routeDefinitions.toARVersionDetailsTab({
              ...versionDetailsTabWithSSCADetailsPathParams,
              ...versionDetailsTabWithOrgAndProjectPathParams
            }),
            // sto with pipeline data
            routeDefinitions.toARVersionDetailsTab({ ...versionDetailsTabWithPipelineDetailsPathParams }),
            // sto with project and pipeline data
            routeDefinitions.toARVersionDetailsTab({
              ...versionDetailsTabWithPipelineDetailsPathParams,
              ...versionDetailsTabWithProjectPathParams
            }),
            // sto with org, project and pipeline data
            routeDefinitions.toARVersionDetailsTab({
              ...versionDetailsTabWithPipelineDetailsPathParams,
              ...versionDetailsTabWithOrgAndProjectPathParams
            })
          ]}>
          <VersionDetailsTabWidget
            onInit={setTab}
            packageType={data.packageType as RepositoryPackageType}
            tab={tab as VersionDetailsTab}
          />
        </RouteProvider>
      </Switch>
    </>
  )
}
