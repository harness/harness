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
import { Icon } from '@harnessio/icons'

import Tag from '@ar/components/Tag/Tag'
import { useStrings } from '@ar/frameworks/strings'
import HeaderTitle from '@ar/components/Header/Title'

import VersionSelector from '../../../components/VersionSelector/VersionSelector'
import ArchitectureSelector from '../ArchitectureSelector/ArchitectureSelector'

interface DockerVersionNameProps {
  name: string
  version: string
  isLatestVersion: boolean
  digest: string
  onChangeVersion: (newVersion: string) => void
  onChangeDigest: (newDigest: string) => void
}
export default function DockerVersionName(props: DockerVersionNameProps): JSX.Element {
  const { name, version, isLatestVersion, digest, onChangeVersion, onChangeDigest } = props
  const { getString } = useStrings()

  return (
    <Layout.Horizontal spacing="small" flex={{ alignItems: 'center', justifyContent: 'flex-start' }}>
      <HeaderTitle>{name}:</HeaderTitle>
      <VersionSelector value={version} onChange={onChangeVersion} />
      <Icon name="chevron-right" size={18} />
      <ArchitectureSelector version={version} value={digest} onChange={onChangeDigest} shouldSelectFirstByDefault />
      {isLatestVersion && <Tag isVersionTag>{getString('tags.latestVersion')}</Tag>}
    </Layout.Horizontal>
  )
}
