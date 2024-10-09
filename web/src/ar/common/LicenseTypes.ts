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

export enum LICENSE_STATE_VALUES {
  ACTIVE = 'ACTIVE',
  DELETED = 'DELETED',
  EXPIRED = 'EXPIRED',
  NOT_STARTED = 'NOT_STARTED'
}

export interface LicenseStoreContextProps {
  readonly licenseInformation: { [key: string]: Record<string, string> } | Record<string, undefined>
  readonly versionMap: { [key: string]: number }
  readonly STO_LICENSE_STATE: LICENSE_STATE_VALUES
  readonly SSCA_LICENSE_STATE: LICENSE_STATE_VALUES
  readonly CI_LICENSE_STATE: LICENSE_STATE_VALUES
  readonly CD_LICENSE_STATE: LICENSE_STATE_VALUES
}
