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

import type { ListArtifactVersion } from '@harnessio/react-har-service-client'

export const mockGenericLatestVersionListTableData: ListArtifactVersion = {
  artifactVersions: [
    {
      fileCount: 10,
      lastModified: '1730978736333',
      name: '1.0.0',
      packageType: 'GENERIC',
      registryIdentifier: '',
      registryPath: '',
      size: '69.56MB'
    }
  ],
  itemCount: 55,
  pageCount: 2,
  pageIndex: 0,
  pageSize: 50
}
