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
import { Menu, MenuItem } from '@blueprintjs/core'
import { getErrorInfoFromErrorObject } from '@harnessio/uicore'
import { useGetAllArtifactVersionsQuery } from '@harnessio/react-har-service-client'

import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import { useStrings } from '@ar/frameworks/strings'
import { getShortDigest } from '@ar/pages/digest-list/utils'
import { useDecodedParams, useGetSpaceRef } from '@ar/hooks'
import type { VersionDetailsPathParams } from '@ar/routes/types'

import type { ListProps } from './type'
import MenuItemLabel from './MenuItemLabel'

import css from './OCIVersionSelector.module.scss'

export default function DigestListSelector({ query, value, onSelect }: ListProps) {
  const registryRef = useGetSpaceRef()
  const params = useDecodedParams<VersionDetailsPathParams>()
  const { getString } = useStrings()
  const { data, isFetching, error } = useGetAllArtifactVersionsQuery({
    registry_ref: registryRef,
    artifact: encodeRef(params.artifactIdentifier),
    queryParams: {
      search_term: query,
      page: 0,
      size: 50
    }
  })
  const digestList = data?.content.data.artifactVersions || []
  return (
    <Menu className={css.menuContainer}>
      {isFetching && <MenuItem className={css.menuItem} disabled text={getString('loading')} />}
      {error && <MenuItem className={css.menuItem} disabled text={getErrorInfoFromErrorObject(error)} />}
      {!isFetching && !error && digestList.length === 0 && (
        <MenuItem className={css.menuItem} disabled text={getString('noResultsFound')} />
      )}
      {digestList.map(each => (
        <MenuItem
          className={css.menuItem}
          key={each.name}
          active={each.name === value.manifest}
          text={<MenuItemLabel>{getShortDigest(each.name)}</MenuItemLabel>}
          onClick={() => onSelect({ manifest: each.name })}
        />
      ))}
    </Menu>
  )
}
