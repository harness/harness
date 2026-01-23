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

import type { ListArtifactScanResponseResponse } from '@harnessio/react-har-service-client'

export const mockData: ListArtifactScanResponseResponse = {
  data: [
    {
      id: '1',
      packageType: 'NPM',
      packageName: 'package1',
      registryName: 'registry1',
      policySetName: 'policy1',
      scanStatus: 'BLOCKED',
      version: '1.0.0',
      registryId: '1',
      policySetRef: 'policy1'
    },
    {
      id: '2',
      packageType: 'NPM',
      packageName: 'package2',
      registryName: 'registry2',
      policySetName: 'policy2',
      scanStatus: 'WARN',
      version: '2.0.0',
      registryId: '2',
      policySetRef: 'policy2'
    }
  ],
  meta: {
    totalCount: 2,
    blockedCount: 1,
    warnCount: 1
  },
  itemCount: 1,
  pageCount: 1,
  pageIndex: 0,
  pageSize: 10
}
