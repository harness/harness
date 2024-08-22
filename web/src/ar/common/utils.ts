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

import type { SyntheticEvent } from 'react'
import type { FormikProps } from 'formik'
import { isEmpty } from 'lodash-es'
import type { FormikFowardRef, RepositoryPackageType } from './types'

export function setFormikRef<T = unknown, U = unknown>(ref: FormikFowardRef<T>, formik: FormikProps<U>): void {
  if (!ref) return

  if (typeof ref === 'function') {
    return
  }

  ref.current = formik as unknown as FormikProps<T>
}

export function getIdentifierStringForBreadcrumb(label: string, value: string): string {
  return `${label}: ${value}`
}

export function killEvent(e: React.MouseEvent<any> | SyntheticEvent<HTMLElement, Event> | undefined): void {
  // do not add preventDefault here, that works odd with checkbox selection
  e?.stopPropagation()
}

export function getPackageTypesForApiQueryParams(packageTypes: RepositoryPackageType[]): string | undefined {
  return isEmpty(packageTypes) ? undefined : packageTypes.join(',')
}
