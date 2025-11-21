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

import type { KVO } from '@harnessio/design-system'
import { useGet } from 'restful-react'
import type { TypesSecret } from 'services/code'

interface useGetSecretListProps {
  space: string
  queryParams: KVO
}

export function useGetSecretList({ space, queryParams }: useGetSecretListProps) {
  return useGet<TypesSecret[]>({
    path: `/api/v1/spaces/${space}/+/secrets`,
    queryParams
  })
}
