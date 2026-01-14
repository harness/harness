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
import { useGetOciArtifactTagsQuery } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import { useDecodedParams, useGetSpaceRef } from '@ar/hooks'
import type { VersionDetailsPathParams } from '@ar/routes/types'

import type { ListProps } from './type'
import MenuItemLabel from './MenuItemLabel'

import css from './OCIVersionSelector.module.scss'

export default function TagListSelector({ query, value, onSelect }: ListProps) {
  const registryRef = useGetSpaceRef()
  const { getString } = useStrings()

  const params = useDecodedParams<VersionDetailsPathParams>()
  const { data, isFetching, error } = useGetOciArtifactTagsQuery({
    registry_ref: registryRef,
    artifact: encodeRef(params.artifactIdentifier),
    queryParams: {
      search_term: query,
      page: 0,
      size: 50
    }
  })

  const tags = data?.content.data.ociArtifactTags || []

  return (
    <Menu className={css.menuContainer}>
      {isFetching && <MenuItem className={css.menuItem} disabled text={getString('loading')} />}
      {error && <MenuItem className={css.menuItem} disabled text={getErrorInfoFromErrorObject(error)} />}
      {!isFetching && !error && tags.length === 0 && (
        <MenuItem className={css.menuItem} disabled text={getString('noResultsFound')} />
      )}
      {tags.map(each => (
        <MenuItem
          className={css.menuItem}
          key={each.name}
          text={<MenuItemLabel>{each.name}</MenuItemLabel>}
          active={each.name === value.tag}
          onClick={() => onSelect({ manifest: each.digest, tag: each.name })}
        />
      ))}
    </Menu>
  )
}
