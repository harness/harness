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

import React, { useMemo } from 'react'

import { useStrings } from '@ar/frameworks/strings'
import MultiSelectDropdownList from '@ar/components/MultiDropdownSelect/MultiDropdownSelect'
import type { RepositoryPackageType } from '@ar/common/types'
import { useGetRepositoryTypes } from '@ar/hooks/useGetRepositoryTypes'

/** Stable default so `useMemo` deps are not invalidated every render when the prop is omitted. */
const EMPTY_EXCLUDE_LIST: RepositoryPackageType[] = []

interface PackageTypeSelectorProps {
  value: RepositoryPackageType[]
  onChange: (val: RepositoryPackageType[]) => void
  /** When set, these package types are omitted from the dropdown and from the controlled selection shown in the UI. */
  excludePackageTypes?: RepositoryPackageType[]
}
export default function PackageTypeSelector(props: PackageTypeSelectorProps): JSX.Element {
  const { value, onChange, excludePackageTypes } = props
  const { getString } = useStrings()
  const repositoryTypes = useGetRepositoryTypes()
  const excluded = excludePackageTypes ?? EMPTY_EXCLUDE_LIST

  const excludeSet = useMemo(() => new Set(excluded), [excluded])

  const items = useMemo(
    () =>
      repositoryTypes
        .filter(each => !each.disabled)
        .filter(each => !excludeSet.has(each.value))
        .map(each => ({ ...each, label: getString(each.label) })),
    [repositoryTypes, excludeSet, getString]
  )

  return (
    <MultiSelectDropdownList
      width={180}
      buttonTestId="package-type-select"
      items={items}
      value={value}
      onSelect={onChange}
      placeholder={getString('repositoryList.selectPackageTypes')}
      allowSearch
    />
  )
}
