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

import type { StringsMap } from '@ar/frameworks/strings'
import { UpstreamRepositoryURLInputSource } from '../../types'

interface RadioGroupItem {
  label: keyof StringsMap
  value: UpstreamRepositoryURLInputSource
}
export const UpstreamURLSourceConfig: Record<UpstreamRepositoryURLInputSource, RadioGroupItem> = {
  [UpstreamRepositoryURLInputSource.Dockerhub]: {
    label: 'upstreamProxyDetails.createForm.source.dockerHub',
    value: UpstreamRepositoryURLInputSource.Dockerhub
  },
  [UpstreamRepositoryURLInputSource.AwsEcr]: {
    label: 'upstreamProxyDetails.createForm.source.ecr',
    value: UpstreamRepositoryURLInputSource.AwsEcr
  },
  [UpstreamRepositoryURLInputSource.Custom]: {
    label: 'upstreamProxyDetails.createForm.source.custom',
    value: UpstreamRepositoryURLInputSource.Custom
  },
  [UpstreamRepositoryURLInputSource.MavenCentral]: {
    label: 'upstreamProxyDetails.createForm.source.mavenCentral',
    value: UpstreamRepositoryURLInputSource.MavenCentral
  },
  [UpstreamRepositoryURLInputSource.NpmJS]: {
    label: 'upstreamProxyDetails.createForm.source.npmjs',
    value: UpstreamRepositoryURLInputSource.NpmJS
  },
  [UpstreamRepositoryURLInputSource.PyPi]: {
    label: 'upstreamProxyDetails.createForm.source.pypi',
    value: UpstreamRepositoryURLInputSource.PyPi
  },
  [UpstreamRepositoryURLInputSource.NugetOrg]: {
    label: 'upstreamProxyDetails.createForm.source.nugetOrg',
    value: UpstreamRepositoryURLInputSource.NugetOrg
  },
  [UpstreamRepositoryURLInputSource.Crates]: {
    label: 'upstreamProxyDetails.createForm.source.crates',
    value: UpstreamRepositoryURLInputSource.Crates
  },
  [UpstreamRepositoryURLInputSource.GoProxy]: {
    label: 'upstreamProxyDetails.createForm.source.goproxy',
    value: UpstreamRepositoryURLInputSource.GoProxy
  },
  [UpstreamRepositoryURLInputSource.HuggingFace]: {
    label: 'upstreamProxyDetails.createForm.source.huggingface',
    value: UpstreamRepositoryURLInputSource.HuggingFace
  },
  [UpstreamRepositoryURLInputSource.Anaconda]: {
    label: 'upstreamProxyDetails.createForm.source.anaconda',
    value: UpstreamRepositoryURLInputSource.Anaconda
  }
}
