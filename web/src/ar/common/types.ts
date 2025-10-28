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

import type { DataTooltipInterface } from '@harnessio/uicore'
import type { FormikContextType, FormikProps } from 'formik'
import type { StringKeys } from '@ar/frameworks/strings'

export interface FormikExtended<T> extends FormikContextType<T> {
  disabled?: boolean
  formName: string
}

export interface FormikContextProps<T> {
  formik?: FormikExtended<T>
  tooltipProps?: DataTooltipInterface
}

export enum Parent {
  OSS = 'OSS',
  Enterprise = 'Enterprise'
}

export enum OCIVersionType {
  TAG = 'TAG',
  DIGEST = 'DIGEST'
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
  PYTHON = 'PYTHON',
  NUGET = 'NUGET',
  RPM = 'RPM',
  GO = 'GO',
  DEBIAN = 'DEBIAN',
  CARGO = 'CARGO',
  ALPINE = 'ALPINE',
  HUGGINGFACE = 'HUGGINGFACE',
  CONDA = 'CONDA'
}

export enum RepositoryConfigType {
  VIRTUAL = 'VIRTUAL',
  UPSTREAM = 'UPSTREAM'
}

export enum PageType {
  Details = 'Details',
  Table = 'Table',
  GlobalList = 'GlobalList'
}

export enum Scanners {
  AQUA_TRIVY = 'AQUA_TRIVY',
  GRYPE = 'GRYPE'
}

export enum EntityScope {
  ACCOUNT = 'ACCOUNT',
  ORG = 'ORG',
  PROJECT = 'PROJECT'
}

export enum RepositoryScopeType {
  NONE = 'none',
  ANCESTORS = 'ancestors',
  DESCENDANTS = 'descendants'
}

export enum RepositoryVisibility {
  PUBLIC = 'PUBLIC',
  PRIVATE = 'PRIVATE'
}

export interface CardSelectOption<T> {
  label: StringKeys
  description: StringKeys
  value: T
}
