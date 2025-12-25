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
import { Layout } from '@harnessio/uicore'

import { Parent } from '@ar/common/types'
import HeaderTitle from '@ar/components/Header/Title'
import { useAppStore, useFeatureFlags } from '@ar/hooks'

import VersionSelector from '../../../components/VersionSelector/VersionSelector'
import OCIVersionSelector from '../../../components/OCIVersionSelector/OCIVersionSelector'

import css from './HelmVersionDetailsHeaderContent.module.scss'

interface HelmVersionNameProps {
  name: string
  version: string
  tag?: string
  onChange: (version: string, tag?: string) => void
}
export default function HelmVersionName(props: HelmVersionNameProps): JSX.Element {
  const { name, version, tag, onChange } = props
  const { parent } = useAppStore()
  const { HAR_ENABLE_UNTAGGED_IMAGES_SUPPORT } = useFeatureFlags()

  return (
    <Layout.Horizontal spacing="medium" className={css.headerWrapper}>
      <HeaderTitle>{name}</HeaderTitle>
      {parent === Parent.OSS || HAR_ENABLE_UNTAGGED_IMAGES_SUPPORT ? (
        <OCIVersionSelector value={{ tag, manifest: version }} onChange={val => onChange(val.manifest, val.tag)} />
      ) : (
        <VersionSelector value={version} onChange={val => onChange(val)} />
      )}
    </Layout.Horizontal>
  )
}
