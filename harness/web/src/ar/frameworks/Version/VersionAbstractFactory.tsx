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

import { isEmpty } from 'lodash-es'
import type { IconName } from '@harnessio/icons'
import type { VersionStep } from './Version'

export interface TypeData {
  name: string
  icon: IconName
  type: string
}

export abstract class VersionAbstractFactory {
  protected abstract type: string

  protected versionTypeBank: Map<string, VersionStep<unknown>>

  constructor() {
    this.versionTypeBank = new Map()
  }

  getType(): string {
    return this.type
  }

  registerStep<T>(step: VersionStep<T>): void {
    this.versionTypeBank.set(step.getPackageType(), step as VersionStep<unknown>)
  }

  deregisterStep(packageType: string): void {
    const deletedStep = this.versionTypeBank.get(packageType)
    if (deletedStep) {
      this.versionTypeBank.delete(packageType)
    }
  }

  getVersionType<T>(packageType?: string): VersionStep<T> | undefined {
    if (packageType && !isEmpty(packageType)) {
      return this.versionTypeBank.get(packageType) as VersionStep<T>
    }
    return
  }
}
