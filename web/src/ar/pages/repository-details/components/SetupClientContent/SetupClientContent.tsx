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
import { FontVariation } from '@harnessio/design-system'
import { Button, ButtonVariation, Layout, Page, Text } from '@harnessio/uicore'
import { type PackageType, useGetClientSetupDetailsQuery } from '@harnessio/react-har-service-client'

import { useGetSpaceRef } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import type { RepositoryPackageType } from '@ar/common/types'
import RepositoryIcon from '@ar/frameworks/RepositoryStep/RepositoryIcon'

import SetupClientSectionContent from './SetupClientSectionContent'

import css from './SetupClientContent.module.scss'

interface SetupClientContentProps {
  onClose: () => void
  repoKey: string
  artifactKey?: string
  versionKey?: string
  packageType: PackageType
}

export default function SetupClientContent(props: SetupClientContentProps): JSX.Element {
  const { onClose, packageType, repoKey } = props
  const { getString } = useStrings()
  const spaceRef = useGetSpaceRef(repoKey)

  const {
    isFetching: loading,
    data,
    error,
    refetch
  } = useGetClientSetupDetailsQuery({
    registry_ref: spaceRef,
    queryParams: {
      artifact: props.artifactKey,
      version: props.versionKey
    }
  })

  const responseData = data?.content.data

  return (
    <Page.Body className={css.pageBody} loading={loading} error={error?.message} retryOnError={() => refetch()}>
      {responseData && (
        <Layout.Vertical>
          <Layout.Horizontal data-testid="setup-client-header" className={css.titleContainer} spacing="medium">
            <RepositoryIcon packageType={packageType as RepositoryPackageType} iconProps={{ size: 28 }} />
            <Text font={{ variation: FontVariation.H3 }}>{responseData.mainHeader}</Text>
          </Layout.Horizontal>
          <Layout.Vertical data-testid="setup-client-body" className={css.contentContainer} spacing="medium">
            <Text font={{ variation: FontVariation.SMALL }}>{responseData.secHeader}</Text>
            {responseData.sections.map((section, index) => (
              <SetupClientSectionContent key={index} section={section} />
            ))}
          </Layout.Vertical>
          <Layout.Horizontal
            data-testid="setup-client-footer"
            padding="xxlarge"
            flex={{ justifyContent: 'flex-start' }}>
            <Button
              variation={ButtonVariation.PRIMARY}
              text={getString('repositoryDetails.clientSetup.done')}
              onClick={onClose}
            />
          </Layout.Horizontal>
        </Layout.Vertical>
      )}
    </Page.Body>
  )
}
