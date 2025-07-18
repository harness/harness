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

import type { Registry, RegistryRequest, UpstreamConfig } from '@harnessio/react-har-service-client'
import type { TypeConfig } from '@ar/pages/repository-details/types'

export enum UpstreamProxyPackageType {
  DOCKER = 'DOCKER',
  HELM = 'HELM',
  GENERIC = 'GENERIC',
  MAVEN = 'MAVEN',
  NPM = 'NPM',
  PYTHON = 'PYTHON',
  NUGET = 'NUGET',
  RPM = 'RPM',
  GO = 'GO',
  DEBIAN = 'DEBIAN',
  CARGO = 'CARGO',
  ALPINE = 'ALPINE'
}

export enum UpstreamRepositoryURLInputSource {
  Dockerhub = 'Dockerhub',
  MavenCentral = 'MavenCentral',
  NpmJS = 'NpmJs',
  AwsEcr = 'AwsEcr',
  Custom = 'Custom',
  PyPi = 'PyPi',
  NugetOrg = 'NugetOrg',
  Crates = 'Crates',
  GoProxy = 'GoProxy'
}

export enum UpstreamProxyAuthenticationMode {
  USER_NAME_AND_PASSWORD = 'UserPassword',
  ACCESS_KEY_AND_SECRET_KEY = 'AccessKeySecretKey',
  ANONYMOUS = 'Anonymous'
}

export type UpstreamRegistryRequest = Omit<RegistryRequest, 'config'> & {
  config: TypeConfig & UpstreamConfig
}

export type UpstreamRegistry = Omit<Registry, 'config'> & {
  config: TypeConfig & UpstreamConfig
}
