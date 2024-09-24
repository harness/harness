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

import React, { useMemo, useRef } from 'react'
import { Expander } from '@blueprintjs/core'
import { defaultTo } from 'lodash-es'
import { flushSync } from 'react-dom'
import { useParams } from 'react-router-dom'
import { useGetArtifactDeploymentsQuery, useGetDockerArtifactManifestsQuery } from '@harnessio/react-har-service-client'
import {
  Button,
  ButtonVariation,
  ExpandingSearchInput,
  ExpandingSearchInputHandle,
  Layout,
  Page
} from '@harnessio/uicore'

import { useGetSpaceRef, useParentHooks } from '@ar/hooks'
import { DEFAULT_PAGE_INDEX, PreferenceScope } from '@ar/constants'
import { useStrings } from '@ar/frameworks/strings'
import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import type { VersionDetailsPathParams } from '@ar/routes/types'
import EnvironmentTypeSelector from '@ar/components/EnvironmentTypeSelector/EnvironmentTypeSelector'

import DockerVersionDeploymentsTable from './components/DeploymentsTable/DeploymentsTable'
import DeploymentOverviewCards from './components/DeploymentOverviewCards/DeploymentOverviewCards'
import {
  useArtifactVersionDeploymentsTableQueryParamOptions,
  type ArtifactVersionDeploymentsTableQueryParams
} from './components/DeploymentsTable/utils'

import css from './DockerVersion.module.scss'

export default function DockerDeploymentsContent() {
  const { useQueryParams, useUpdateQueryParams, usePreferenceStore } = useParentHooks()
  const { updateQueryParams } = useUpdateQueryParams<Partial<ArtifactVersionDeploymentsTableQueryParams>>()
  const queryParamOptions = useArtifactVersionDeploymentsTableQueryParamOptions()
  const queryParams = useQueryParams<ArtifactVersionDeploymentsTableQueryParams>(queryParamOptions)
  const { environmentTypes, searchTerm } = queryParams
  const searchRef = useRef({} as ExpandingSearchInputHandle)
  const { getString } = useStrings()
  const registryRef = useGetSpaceRef()
  const params = useParams<VersionDetailsPathParams>()

  const { preference: sortingPreference, setPreference: setSortingPreference } = usePreferenceStore<string | undefined>(
    PreferenceScope.USER,
    'ArtifactVersionDeploymentsList'
  )
  const sort = useMemo(
    () => (sortingPreference ? JSON.parse(sortingPreference) : queryParams.sort),
    [queryParams.sort, sortingPreference]
  )

  const { data, isFetching, error, refetch } = useGetArtifactDeploymentsQuery({
    registry_ref: registryRef,
    artifact: encodeRef(params.artifactIdentifier),
    version: params.versionIdentifier,
    queryParams: {
      search_term: searchTerm,
      env_type: environmentTypes.length === 1 ? environmentTypes[0] : undefined
    }
  })

  const { data: manifestsData } = useGetDockerArtifactManifestsQuery({
    registry_ref: registryRef,
    artifact: encodeRef(params.artifactIdentifier),
    version: params.versionIdentifier
  })

  const handleClearFilters = (): void => {
    flushSync(searchRef.current.clear)
    updateQueryParams({
      page: 0,
      searchTerm: '',
      environmentTypes: []
    })
  }

  const responseData = data?.content.data
  const hasFilter = searchTerm || environmentTypes?.length

  return (
    <Layout.Vertical padding="large" spacing="medium">
      {responseData && (
        <DeploymentOverviewCards
          artifactDetails={{
            artifactName: params.artifactIdentifier,
            version: params.versionIdentifier,
            digests: defaultTo(manifestsData?.content.data.manifests, [])
          }}
          deploymentStats={responseData?.deploymentsStats}
        />
      )}
      <div className={css.subHeaderItems}>
        <EnvironmentTypeSelector
          value={environmentTypes}
          onChange={val => {
            updateQueryParams({ environmentTypes: val, page: DEFAULT_PAGE_INDEX })
          }}
        />
        <Expander />
        <ExpandingSearchInput
          alwaysExpanded
          width={200}
          placeholder={getString('search')}
          onChange={text => {
            updateQueryParams({ searchTerm: text || undefined, page: DEFAULT_PAGE_INDEX })
          }}
          defaultValue={searchTerm}
          ref={searchRef}
        />
      </div>
      <Page.Body
        className={css.deploymentTablePageBody}
        loading={isFetching}
        error={error?.message}
        retryOnError={() => refetch()}
        noData={{
          when: () => !responseData?.deployments.itemCount,
          icon: 'thinner-code-repos',
          // image: getEmptyStateIllustration(hasFilter, module),
          messageTitle: hasFilter
            ? getString('noResultsFound')
            : getString('versionDetails.deploymentsTable.noDeploymentsTitle'),
          button: hasFilter ? (
            <Button text={getString('clearFilters')} variation={ButtonVariation.LINK} onClick={handleClearFilters} />
          ) : (
            <></>
          )
        }}>
        {responseData && (
          <DockerVersionDeploymentsTable
            data={responseData}
            gotoPage={pageNumber => updateQueryParams({ page: pageNumber })}
            onPageSizeChange={newSize => updateQueryParams({ size: newSize, page: DEFAULT_PAGE_INDEX })}
            setSortBy={sortArray => {
              setSortingPreference(JSON.stringify(sortArray))
              updateQueryParams({ sort: sortArray, page: DEFAULT_PAGE_INDEX })
            }}
            sortBy={sort}
          />
        )}
      </Page.Body>
    </Layout.Vertical>
  )
}
