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

import React, { useCallback, useContext } from 'react'
import type { FormikProps } from 'formik'
import { Expander } from '@blueprintjs/core'
import { Button, ButtonVariation, Container, Layout, Tab, Tabs } from '@harnessio/uicore'

import type { RepositoryDetailsPathParams } from '@ar/routes/types'
import { useDecodedParams, useParentComponents, useParentHooks } from '@ar/hooks'
import { PermissionIdentifier, ResourceType } from '@ar/common/permissionTypes'
import { useStrings } from '@ar/frameworks/strings'
import type { RepositoryConfigType, RepositoryPackageType } from '@ar/common/types'
import RepositoryDetailsHeaderWidget from '@ar/frameworks/RepositoryStep/RepositoryDetailsHeaderWidget'
import RepositoryConfigurationFormWidget from '@ar/frameworks/RepositoryStep/RepositoryConfigurationFormWidget'

import type { Repository } from './types'
import { RepositoryDetailsTab } from './constants'
import { RepositoryProviderContext } from './context/RepositoryProvider'
import RegistryArtifactListPage from '../artifact-list/RegistryArtifactListPage'
import css from './RepositoryDetailsPage.module.scss'

export default function RepositoryDetails(): JSX.Element | null {
  const { useUpdateQueryParams, useQueryParams } = useParentHooks()
  const { updateQueryParams } = useUpdateQueryParams()
  const { RbacButton } = useParentComponents()
  const { getString } = useStrings()
  const { repositoryIdentifier } = useDecodedParams<RepositoryDetailsPathParams>()
  const { tab: selectedTabId = RepositoryDetailsTab.PACKAGES } = useQueryParams<{ tab: RepositoryDetailsTab }>()
  const stepRef = React.useRef<FormikProps<unknown> | null>(null)

  const { isDirty, data, isReadonly, isUpdating } = useContext(RepositoryProviderContext)

  const { config } = data as Repository
  const { type } = config

  const handleTabChange = useCallback(
    (nextTab: RepositoryDetailsTab): void => {
      updateQueryParams({ tab: nextTab })
    },
    [updateQueryParams]
  )

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
          permission: PermissionIdentifier.EDIT_ARTIFACT_REGISTRY,
          resource: {
            resourceType: ResourceType.ARTIFACT_REGISTRY,
            resourceIdentifier: repositoryIdentifier
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

  if (!data) return null

  return (
    <>
      <RepositoryDetailsHeaderWidget
        data={data}
        packageType={data.packageType as RepositoryPackageType}
        type={data.config.type as RepositoryConfigType}
      />
      <Container className={css.tabsContainer}>
        <Tabs id="repositoryTabDetails" selectedTabId={selectedTabId} onChange={handleTabChange}>
          <Tab
            id={RepositoryDetailsTab.PACKAGES}
            title={getString('repositoryDetails.tabs.packages')}
            panel={
              <Container>
                <RegistryArtifactListPage pageBodyClassName={css.packagesPageBody} />
              </Container>
            }
          />
          <Tab
            id={RepositoryDetailsTab.CONFIGURATION}
            title={getString('repositoryDetails.tabs.configuration')}
            panel={
              <RepositoryConfigurationFormWidget
                packageType={data.packageType as RepositoryPackageType}
                type={type as RepositoryConfigType}
                ref={stepRef}
                readonly={isReadonly}
              />
            }
          />
          <Expander />
          {selectedTabId === RepositoryDetailsTab.CONFIGURATION && renderActionBtns()}
        </Tabs>
      </Container>
    </>
  )
}
