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

import Tag from '@ar/components/Tag/Tag'
import { Parent } from '@ar/common/types'
import { useStrings } from '@ar/frameworks/strings'
import HeaderTitle from '@ar/components/Header/Title'
import { useAppStore, useFeatureFlags } from '@ar/hooks'

import VersionSelector from '../../../components/VersionSelector/VersionSelector'
import ArchitectureSelector from '../ArchitectureSelector/ArchitectureSelector'
import OCIVersionSelector from '../../../components/OCIVersionSelector/OCIVersionSelector'

import css from './DockerVersionName.module.scss'

interface DockerVersionNameProps {
  name: string
  version: string
  isLatestVersion: boolean
  digest: string
  tag?: string
  onChangeVersion: (version: string, tag?: string) => void
  onChangeDigest: (newDigest: string) => void
}
export default function DockerVersionName(props: DockerVersionNameProps): JSX.Element {
  const { name, version, tag, isLatestVersion, digest, onChangeVersion, onChangeDigest } = props
  const { getString } = useStrings()
  const { parent } = useAppStore()
  const { HAR_ENABLE_UNTAGGED_IMAGES_SUPPORT } = useFeatureFlags()

  return (
    <Layout.Horizontal spacing="medium" className={css.headerWrapper}>
      <HeaderTitle>{name}</HeaderTitle>
      {parent === Parent.OSS || HAR_ENABLE_UNTAGGED_IMAGES_SUPPORT ? (
        <OCIVersionSelector
          value={{ tag, manifest: version }}
          onChange={val => onChangeVersion(val.manifest, val.tag)}
        />
      ) : (
        <VersionSelector value={version} onChange={val => onChangeVersion(val)} />
      )}
      <ArchitectureSelector version={version} value={digest} onChange={onChangeDigest} shouldSelectFirstByDefault />
      {isLatestVersion && <Tag isVersionTag>{getString('tags.latestVersion')}</Tag>}
    </Layout.Horizontal>
  )
}
