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

import React, { useContext, useMemo, useState } from 'react'
import type { FormikProps } from 'formik'
import { Expander } from '@blueprintjs/core'
import { Redirect, Switch, useHistory, useParams } from 'react-router-dom'
import { Button, ButtonVariation, Container, Layout, Tab, Tabs } from '@harnessio/uicore'
import type { PackageType } from '@harnessio/react-har-service-client'

import { Parent } from '@ar/common/types'
import { useStrings } from '@ar/frameworks/strings'
import type { ArtifactDetailsPathParams } from '@ar/routes/types'
import versionFactory from '@ar/frameworks/Version/VersionFactory'
import TabsContainer from '@ar/components/TabsContainer/TabsContainer'
import RouteProvider from '@ar/components/RouteProvider/RouteProvider'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'
import PropertiesFormContent from '@ar/components/PropertiesForm/PropertiesFormContent'
import { artifactDetailsPathProps, versionDetailsPathParams } from '@ar/routes/RouteDestinations'
import { useAppStore, useFeatureFlags, useGetRepositoryListViewType, useParentComponents, useRoutes } from '@ar/hooks'

import VersionListPage from '../version-list/VersionListPage'
import { ArtifactProviderContext } from './context/ArtifactProvider'
import { ArtifactDetailsTab, ArtifactDetailsTabs } from './constants'
import OSSVersionDetailsPage from '../version-details/OSSVersionDetailsPage'

import css from './ArtifactDetails.module.scss'
import formContent from '@ar/pages/repository-details/components/FormContent/FormContent.module.scss'

function ArtifactDetails(): JSX.Element {
  const [activeTab, setActiveTab] = useState(ArtifactDetailsTab.VERSIONS)
  const pathParams = useParams<ArtifactDetailsPathParams>()
  const { getString } = useStrings()
  const history = useHistory()
  const routeDefinitions = useRoutes(true)
  const routes = useRoutes()
  const featureFlags = useFeatureFlags()
  const { parent } = useAppStore()
  const { RbacButton } = useParentComponents()
  const { data, isDirty, isUpdating, isReadonly, setIsDirty, setIsUpdating } = useContext(ArtifactProviderContext)
  const repositoryListViewType = useGetRepositoryListViewType()
  const stepRef = React.useRef<FormikProps<unknown> | null>(null)

  const artifactTabs = useMemo(() => {
    if (!data) return []
    const versionType = versionFactory.getVersionType(data?.packageType)
    const tabs = versionType?.getSupportedArtifactTabs() || []
    return ArtifactDetailsTabs.filter(each => tabs.includes(each.value))
      .filter(each => !each.featureFlag || featureFlags[each.featureFlag])
      .filter(each => !each.mode || each.mode === repositoryListViewType)
      .filter(each => !each.parent || each.parent === parent)
  }, [data, featureFlags, parent, repositoryListViewType])

  const activeTabConfig = useMemo(() => artifactTabs.find(each => each.value === activeTab), [artifactTabs, activeTab])

  const handleTabChange = (nextTab: ArtifactDetailsTab): void => {
    setActiveTab(nextTab)
    switch (nextTab) {
      case ArtifactDetailsTab.VERSIONS:
        history.push(routes.toARArtifactVersions({ ...pathParams }))
        break
      case ArtifactDetailsTab.METADATA:
        history.push(routes.toARArtifactProperties({ ...pathParams }))
        break
    }
  }

  const handleSubmitForm = (): void => {
    stepRef.current?.submitForm()
  }

  const handleResetForm = (): void => {
    stepRef.current?.resetForm()
  }

  const renderActionBtns = (): JSX.Element => (
    <Layout.Horizontal className={css.btnContainer}>
      <RbacButton
        text={getString('save')}
        className={css.saveButton}
        variation={ButtonVariation.PRIMARY}
        onClick={handleSubmitForm}
        disabled={!isDirty || isUpdating}
        permission={{
          permission: PermissionIdentifier.DOWNLOAD_ARTIFACT,
          resource: {
            resourceType: ResourceType.ARTIFACT_REGISTRY,
            resourceIdentifier: pathParams.repositoryIdentifier
          }
        }}
      />
      <Button
        className={css.discardBtn}
        variation={ButtonVariation.SECONDARY}
        text={getString('discard')}
        onClick={handleResetForm}
        disabled={!isDirty}
      />
    </Layout.Horizontal>
  )

  return (
    <>
      <TabsContainer className={css.tabsContainer}>
        <Tabs id="artifactTabDetails" selectedTabId={activeTab} onChange={handleTabChange}>
          {artifactTabs.map(each => (
            <Tab key={each.value} id={each.value} title={getString(each.label)} />
          ))}
          <Expander />
          {activeTabConfig?.supportActions && renderActionBtns()}
        </Tabs>
      </TabsContainer>
      <Switch>
        <RouteProvider
          exact
          path={routeDefinitions.toARArtifactVersions({ ...artifactDetailsPathProps })}
          isPublic
          onLoad={() => {
            setActiveTab(ArtifactDetailsTab.VERSIONS)
          }}>
          <VersionListPage packageType={data?.packageType as PackageType} />
        </RouteProvider>
        <RouteProvider
          exact
          isPublic
          path={routeDefinitions.toARArtifactProperties({ ...artifactDetailsPathProps })}
          onLoad={() => {
            setActiveTab(ArtifactDetailsTab.METADATA)
          }}>
          <Container padding="xlarge">
            <PropertiesFormContent
              readonly={isReadonly}
              ref={stepRef}
              setIsDirty={setIsDirty}
              setIsUpdating={setIsUpdating}
              repositoryIdentifier={pathParams.repositoryIdentifier}
              artifactIdentifier={pathParams.artifactIdentifier}
              className={formContent.cardContainer}
            />
          </Container>
        </RouteProvider>
        {parent === Parent.OSS && (
          <RouteProvider isPublic path={routes.toARVersionDetails({ ...versionDetailsPathParams })}>
            <OSSVersionDetailsPage />
          </RouteProvider>
        )}
        <RouteProvider isPublic exact path={routeDefinitions.toARArtifactDetails({ ...artifactDetailsPathProps })}>
          <Redirect to={routes.toARArtifactVersions({ ...pathParams })} />
        </RouteProvider>
      </Switch>
    </>
  )
}

export default ArtifactDetails
