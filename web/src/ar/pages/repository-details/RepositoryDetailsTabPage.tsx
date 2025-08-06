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
import { useEffect } from 'react'
import { Layout, Text } from '@harnessio/uicore'
import type { FormikProps } from 'formik'

import { useDecodedParams, useFeatureFlags } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import type { RepositoryDetailsTabPathParams } from '@ar/routes/types'
import type { RepositoryConfigType, RepositoryPackageType } from '@ar/common/types'
import RepositoryConfigurationFormWidget from '@ar/frameworks/RepositoryStep/RepositoryConfigurationFormWidget'

import { RepositoryDetailsTab } from './constants'
import WebhookListPage from '../webhook-list/WebhookListPage'
import { RepositoryProviderContext } from './context/RepositoryProvider'
import RegistryArtifactListPage from '../artifact-list/RegistryArtifactListPage'
import ManageRepositoryMetadata from './components/ManageRepositoryMetadata/ManageRepositoryMetadata'

import css from './RepositoryDetailsPage.module.scss'

interface RepositoryDetailsTabPageProps {
  onInit: (tab: RepositoryDetailsTab) => void
  stepRef: React.RefObject<FormikProps<unknown>>
}

export default function RepositoryDetailsTabPage(props: RepositoryDetailsTabPageProps): JSX.Element {
  const { onInit, stepRef } = props
  const { getString } = useStrings()
  const { tab } = useDecodedParams<RepositoryDetailsTabPathParams>()
  const { data, isReadonly } = useContext(RepositoryProviderContext)
  const { HAR_CUSTOM_METADATA_ENABLED } = useFeatureFlags()

  useEffect(() => {
    onInit(tab)
  }, [tab])

  switch (tab) {
    case RepositoryDetailsTab.PACKAGES:
    case RepositoryDetailsTab.DATASETS:
    case RepositoryDetailsTab.MODELS:
      return <RegistryArtifactListPage pageBodyClassName={css.packagesPageBody} />
    case RepositoryDetailsTab.CONFIGURATION:
      return (
        <Layout.Horizontal
          padding="xlarge"
          spacing="medium"
          flex={{ justifyContent: 'space-between', alignItems: 'flex-start' }}>
          <RepositoryConfigurationFormWidget
            packageType={data?.packageType as RepositoryPackageType}
            type={data?.config.type as RepositoryConfigType}
            ref={stepRef}
            readonly={isReadonly}
          />
          {HAR_CUSTOM_METADATA_ENABLED && <ManageRepositoryMetadata />}
        </Layout.Horizontal>
      )
    case RepositoryDetailsTab.WEBHOOKS:
      return <WebhookListPage />
    default:
      return <Text intent="warning">{getString('stepNotFound')}</Text>
  }
}
