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
import type { RepositoryStep } from './Repository'

export abstract class RepositoryAbstractFactory {
  protected abstract type: string

  protected repositoryTypeBank: Map<string, RepositoryStep<unknown>>

  constructor() {
    this.repositoryTypeBank = new Map()
  }

  getType(): string {
    return this.type
  }

  registerStep<T>(step: RepositoryStep<T>): void {
    this.repositoryTypeBank.set(step.getPackageType(), step as RepositoryStep<unknown>)
  }

  deregisterStep(packageType: string): void {
    const deletedStep = this.repositoryTypeBank.get(packageType)
    if (deletedStep) {
      this.repositoryTypeBank.delete(packageType)
    }
  }

  getRepositoryType<T>(packageType?: string): RepositoryStep<T> | undefined {
    if (packageType && !isEmpty(packageType)) {
      return this.repositoryTypeBank.get(packageType) as RepositoryStep<T>
    }
    return
  }
}
