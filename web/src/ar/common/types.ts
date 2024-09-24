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

import type { FormikProps } from 'formik'

export enum Parent {
  OSS = 'OSS',
  Enterprise = 'Enterprise'
}

export type FormikRef<T> = Pick<FormikProps<T>, 'submitForm' | 'errors'>

export type FormikFowardRef<T = unknown> =
  | ((instance: FormikRef<T> | null) => void)
  | React.MutableRefObject<FormikRef<T> | null>
  | null

export enum EnvironmentType {
  Prod = 'Production',
  NonProd = 'PreProduction'
}

export enum RepositoryPackageType {
  DOCKER = 'DOCKER',
  HELM = 'HELM',
  GENERIC = 'GENERIC',
  MAVEN = 'MAVEN',
  NPM = 'NPM',
  GRADLE = 'GRADLE',
  PYPI = 'PYPI',
  NUGET = 'NUGET'
}

export enum RepositoryConfigType {
  VIRTUAL = 'VIRTUAL',
  UPSTREAM = 'UPSTREAM'
}

export enum PageType {
  Details = 'Details',
  Table = 'Table'
}
