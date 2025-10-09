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

import React from 'react'
import { Button, ButtonProps } from '@harnessio/uicore'

import type { PermissionsRequest } from '@ar/MFEAppTypes'
import type { PermissionIdentifier } from '@ar/common/permissionTypes'

export interface RbacButtonProps extends ButtonProps {
  permission?: Omit<PermissionsRequest, 'permissions'> & { permission: PermissionIdentifier }
}

export default function RbacButton(props: RbacButtonProps) {
  return <Button {...props} />
}
