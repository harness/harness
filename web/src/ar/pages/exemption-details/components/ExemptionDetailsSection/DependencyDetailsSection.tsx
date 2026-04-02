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
import { Link } from 'react-router-dom'
import { Container, Layout } from '@harnessio/uicore'
import type { FirewallExceptionResponseV3 } from '@harnessio/react-har-service-client'

import { useRoutes } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import RepositoryIcon from '@ar/frameworks/RepositoryStep/RepositoryIcon'

import type { RepositoryPackageType } from '@ar/common/types'
import { Label, VersionActionBtn } from './Components'

import css from './ExemptionDetailsSection.module.scss'

interface DependencyDetailsSectionProps {
  data: FirewallExceptionResponseV3
}

function DependencyDetailsSection({ data }: DependencyDetailsSectionProps) {
  const { getString } = useStrings()
  const routes = useRoutes()
  return (
    <Container className={css.gridContainer}>
      {/* Package Name Row */}
      <Label>{getString('exemptionDetails.cards.section1.packageName')}</Label>
      <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
        <RepositoryIcon iconProps={{ size: 16 }} packageType={data.packageType as RepositoryPackageType} />
        <Link
          to={routes.toARRedirect({
            packageType: data.packageType as RepositoryPackageType,
            registryId: data.registryName || '',
            artifactId: data.packageName
          })}>
          {data.packageName}
        </Link>
      </Layout.Horizontal>
      {/* Version List Row */}
      <Label>{getString('exemptionDetails.cards.section1.versions')}</Label>
      <Layout.Vertical flex={{ justifyContent: 'flex-start', alignItems: 'flex-start' }}>
        {data.versionList?.map(each => {
          return (
            <VersionActionBtn key={each} scanId={data.versionScanMap?.[each]}>
              {each}
            </VersionActionBtn>
          )
        })}
      </Layout.Vertical>
      {/* Upstream Registry */}
      <Label>{getString('exemptionDetails.cards.section1.upstreamProxy')}</Label>
      <Link
        to={routes.toARRedirect({
          packageType: data.packageType as RepositoryPackageType,
          registryId: data.registryName || ''
        })}>
        {data.registryName}
      </Link>
    </Container>
  )
}

export default DependencyDetailsSection
